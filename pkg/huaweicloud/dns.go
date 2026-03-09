package huaweicloud

import (
	"fmt"
	"strings"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	dnsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// DNSClient is a wrapper around Huawei Cloud DNS SDK client
type DNSClient struct {
	client     *dns.DnsClient
	regionID   string
	projectID  string
	zoneName   string
	zoneID     string
}

// NewDNSClient creates a new Huawei Cloud DNS client
func NewDNSClient(regionName, projectID, ak, sk, zoneName string) (*DNSClient, error) {
	// Create auth credential
	auth, err := basic.NewCredentialsBuilder().
		WithAk(ak).
		WithSk(sk).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials: %w", err)
	}

	// Create region
	regionID, err := getRegionID(regionName)
	if err != nil {
		return nil, fmt.Errorf("invalid region %s: %w", regionName, err)
	}

	// Create HTTP client
	httpClient, err := dns.DnsClientBuilder().
		WithRegion(region.NewRegion(regionID, regionID)).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create DNS client
	client := dns.NewDnsClient(httpClient)

	d := &DNSClient{
		client:    client,
		regionID:  regionID,
		projectID: projectID,
		zoneName:  zoneName,
	}

	// Get zone ID
	zoneID, err := d.getZoneID()
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}
	d.zoneID = zoneID

	return d, nil
}

// getRegionID converts region name to region using SDK's region lookup
func getRegionID(region string) (string, error) {
	// Use SDK's built-in region lookup for DNS service
	dnsRegion, err := dnsregion.SafeValueOf(region)
	if err != nil {
		return "", fmt.Errorf("invalid region %s: %w", region, err)
	}
	return dnsRegion.Id, nil
}

// getZoneID retrieves the zone ID from zone name
func (d *DNSClient) getZoneID() (string, error) {
	// List all public zones
	request := &model.ListPublicZonesRequest{}
	zoneType := "public"
	request.Type = &zoneType
	limit := int32(100)
	request.Limit = &limit

	response, err := d.client.ListPublicZones(request)
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	if response.Zones == nil || len(*response.Zones) == 0 {
		return "", fmt.Errorf("no zones found")
	}

	// Find matching zone
	for _, zone := range *response.Zones {
		// Zone names may or may not have trailing dot, handle both
		zoneName := strings.TrimSuffix(*zone.Name, ".")
		if zoneName == d.zoneName || zoneName == d.zoneName+"." {
			return *zone.Id, nil
		}
	}

	return "", fmt.Errorf("zone %s not found", d.zoneName)
}

// CreateTXTRecord creates a TXT record for ACME challenge
func (d *DNSClient) CreateTXTRecord(fqdn, value string, ttl int) error {
	// Extract the record name (remove zone name suffix)
	recordName := d.extractRecordName(fqdn)

	// Huawei Cloud TXT record values must be quoted
	quotedValue := fmt.Sprintf("\"%s\"", value)

	// Create description
	description := "ACME challenge record"

	ttlValue := int32(ttl)

	request := &model.CreateRecordSetRequest{
		ZoneId: d.zoneID,
		Body: &model.CreateRecordSetRequestBody{
			Name:        recordName,
			Type:        "TXT",
			Ttl:         &ttlValue,
			Records:     []string{quotedValue},
			Description: &description,
		},
	}

	_, err := d.client.CreateRecordSet(request)
	if err != nil {
		return fmt.Errorf("failed to create TXT record: %w", err)
	}

	return nil
}

// DeleteTXTRecord deletes a TXT record by matching the value
func (d *DNSClient) DeleteTXTRecord(fqdn, value string) error {
	// Get all TXT records for this FQDN
	recordName := d.extractRecordName(fqdn)
	records, err := d.listTXTRecords(recordName)
	if err != nil {
		return fmt.Errorf("failed to list TXT records: %w", err)
	}

	// Find and delete the record matching the value
	for _, record := range records {
		if record.Records != nil && len(*record.Records) > 0 {
			// Remove quotes from stored value for comparison
			storedValue := strings.Trim((*record.Records)[0], `"`)
			if storedValue == value {
				return d.deleteRecordByID(*record.Id)
			}
		}
	}

	// If we didn't find the exact record, return nil (idempotent)
	return nil
}

// listTXTRecords lists all TXT records for a given record name
func (d *DNSClient) listTXTRecords(recordName string) ([]model.ListRecordSets, error) {
	request := &model.ListRecordSetsByZoneRequest{
		ZoneId: d.zoneID,
	}

	response, err := d.client.ListRecordSetsByZone(request)
	if err != nil {
		return nil, fmt.Errorf("failed to list record sets: %w", err)
	}

	if response.Recordsets == nil {
		return []model.ListRecordSets{}, nil
	}

	// Filter for TXT records matching the record name
	var result []model.ListRecordSets
	for _, record := range *response.Recordsets {
		if record.Type != nil && *record.Type == "TXT" &&
			record.Name != nil && *record.Name == recordName {
			result = append(result, record)
		}
	}

	return result, nil
}

// deleteRecordByID deletes a record by its ID
func (d *DNSClient) deleteRecordByID(recordID string) error {
	request := &model.DeleteRecordSetRequest{
		ZoneId:     d.zoneID,
		RecordsetId: recordID,
	}

	_, err := d.client.DeleteRecordSet(request)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// extractRecordName extracts the record name from FQDN by removing zone suffix
func (d *DNSClient) extractRecordName(fqdn string) string {
	// Ensure both have consistent trailing dots
	fqdn = strings.TrimSuffix(fqdn, ".")
	zoneName := strings.TrimSuffix(d.zoneName, ".")

	// Check if fqdn ends with zone name
	if strings.HasSuffix(fqdn, "."+zoneName) {
		return fqdn
	}

	// Return as-is if pattern doesn't match
	return fqdn
}
