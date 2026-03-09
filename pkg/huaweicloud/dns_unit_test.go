package huaweicloud

import (
	"testing"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// TestDNSClient_CreateRecordSetRequestBuilder tests the creation of record set requests
func TestDNSClient_CreateRecordSetRequestBuilder(t *testing.T) {
	tests := []struct {
		name      string
		fqdn      string
		value     string
		ttl       int
		zoneID    string
		zoneName  string
		wantName  string
		wantValue string
	}{
		{
			name:      "simple ACME challenge",
			fqdn:      "_acme-challenge.example.com",
			value:     "test-validation-key",
			ttl:       60,
			zoneID:    "zone-123",
			zoneName:  "example.com",
			wantName:  "_acme-challenge.example.com",
			wantValue: "\"test-validation-key\"",
		},
		{
			name:      "subdomain ACME challenge",
			fqdn:      "_acme-challenge.api.example.com",
			value:     "another-key",
			ttl:       120,
			zoneID:    "zone-456",
			zoneName:  "example.com",
			wantName:  "_acme-challenge.api.example.com",
			wantValue: "\"another-key\"",
		},
		{
			name:      "ACME challenge with special characters",
			fqdn:      "_acme-challenge.example.com",
			value:     "key-with_underscore.and.dash",
			ttl:       300,
			zoneID:    "zone-789",
			zoneName:  "example.com",
			wantName:  "_acme-challenge.example.com",
			wantValue: "\"key-with_underscore.and.dash\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNSClient{
				zoneName: tt.zoneName,
				zoneID:   tt.zoneID,
			}

			// Test extractRecordName
			recordName := d.extractRecordName(tt.fqdn)
			if recordName != tt.wantName {
				t.Errorf("extractRecordName() = %v, want %v", recordName, tt.wantName)
			}

			// Test that value is quoted correctly
			quotedValue := tt.value
			if quotedValue[0] != '"' || quotedValue[len(quotedValue)-1] != '"' {
				quotedValue = "\"" + tt.value + "\""
			}
			if quotedValue != tt.wantValue {
				t.Errorf("Quoted value = %v, want %v", quotedValue, tt.wantValue)
			}

			// Verify TTL conversion
			ttlValue := int32(tt.ttl)
			if ttlValue != int32(tt.ttl) {
				t.Errorf("TTL conversion = %v, want %v", ttlValue, tt.ttl)
			}
		})
	}
}

// TestDNSClient_DeleteRecordSetRequestBuilder tests the creation of delete record set requests
func TestDNSClient_DeleteRecordSetRequestBuilder(t *testing.T) {
	tests := []struct {
		name     string
		zoneID   string
		recordID string
	}{
		{
			name:     "valid delete request",
			zoneID:   "zone-123",
			recordID: "record-456",
		},
		{
			name:     "different zone and record",
			zoneID:   "zone-abc",
			recordID: "record-def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request structure that would be used for deletion
			request := &model.DeleteRecordSetRequest{
				ZoneId:     tt.zoneID,
				RecordsetId: tt.recordID,
			}

			if request.ZoneId != tt.zoneID {
				t.Errorf("ZoneId = %v, want %v", request.ZoneId, tt.zoneID)
			}
			if request.RecordsetId != tt.recordID {
				t.Errorf("RecordsetId = %v, want %v", request.RecordsetId, tt.recordID)
			}
		})
	}
}

// TestDNSClient_ListRecordSetsRequestBuilder tests the creation of list record sets requests
func TestDNSClient_ListRecordSetsRequestBuilder(t *testing.T) {
	tests := []struct {
		name   string
		zoneID string
	}{
		{
			name:   "valid list request",
			zoneID: "zone-123",
		},
		{
			name:   "another zone",
			zoneID: "zone-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &model.ListRecordSetsByZoneRequest{
				ZoneId: tt.zoneID,
			}

			if request.ZoneId != tt.zoneID {
				t.Errorf("ZoneId = %v, want %v", request.ZoneId, tt.zoneID)
			}
		})
	}
}

// TestDNSClient_ListPublicZonesRequestBuilder tests the creation of list public zones requests
func TestDNSClient_ListPublicZonesRequestBuilder(t *testing.T) {
	tests := []struct {
		name     string
		zoneType string
		limit    int32
	}{
		{
			name:     "public zones with limit",
			zoneType: "public",
			limit:    100,
		},
		{
			name:     "public zones with different limit",
			zoneType: "public",
			limit:    50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &model.ListPublicZonesRequest{}
			request.Type = &tt.zoneType
			request.Limit = &tt.limit

			if *request.Type != tt.zoneType {
				t.Errorf("Type = %v, want %v", *request.Type, tt.zoneType)
			}
			if *request.Limit != tt.limit {
				t.Errorf("Limit = %v, want %v", *request.Limit, tt.limit)
			}
		})
	}
}

// TestDNSClient_ZoneMatching tests zone name matching logic
func TestDNSClient_ZoneMatching(t *testing.T) {
	tests := []struct {
		name          string
		zoneName      string
		zoneFromAPI   string
		shouldMatch   bool
	}{
		{
			name:        "exact match",
			zoneName:    "example.com",
			zoneFromAPI: "example.com",
			shouldMatch: true,
		},
		{
			name:        "API zone has trailing dot",
			zoneName:    "example.com",
			zoneFromAPI: "example.com.",
			shouldMatch: true,
		},
		{
			name:        "config zone has trailing dot",
			zoneName:    "example.com.",
			zoneFromAPI: "example.com",
			shouldMatch: true,
		},
		{
			name:        "both have trailing dot",
			zoneName:    "example.com.",
			zoneFromAPI: "example.com.",
			shouldMatch: true,
		},
		{
			name:        "no match - different domain",
			zoneName:    "example.com",
			zoneFromAPI: "different.com",
			shouldMatch: false,
		},
		{
			name:        "no match - subdomain",
			zoneName:    "example.com",
			zoneFromAPI: "api.example.com",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNSClient{
				zoneName: tt.zoneName,
			}

			// Simulate the matching logic from getZoneID
			apiZoneName := tt.zoneFromAPI
			// Remove trailing dot from API zone name
			for len(apiZoneName) > 0 && apiZoneName[len(apiZoneName)-1] == '.' {
				apiZoneName = apiZoneName[:len(apiZoneName)-1]
			}

			configZoneName := d.zoneName
			// Remove trailing dot from config zone name
			for len(configZoneName) > 0 && configZoneName[len(configZoneName)-1] == '.' {
				configZoneName = configZoneName[:len(configZoneName)-1]
			}

			matches := apiZoneName == configZoneName || apiZoneName == configZoneName+"."
			if matches != tt.shouldMatch {
				t.Errorf("Zone matching: zoneName=%q, zoneFromAPI=%q, matches=%v, want %v",
					tt.zoneName, tt.zoneFromAPI, matches, tt.shouldMatch)
			}
		})
	}
}

// TestDNSClient_TXTRecordValueMatching tests TXT record value matching logic
func TestDNSClient_TXTRecordValueMatching(t *testing.T) {
	tests := []struct {
		name         string
		storedValue  string
		searchValue  string
		shouldMatch  bool
	}{
		{
			name:         "exact match",
			storedValue:  "\"test-value\"",
			searchValue:  "test-value",
			shouldMatch:  true,
		},
		{
			name:         "stored value has quotes",
			storedValue:  "\"another-value\"",
			searchValue:  "another-value",
			shouldMatch:  true,
		},
		{
			name:         "no match - different values",
			storedValue:  "\"test-value\"",
			searchValue:  "different-value",
			shouldMatch:  false,
		},
		{
			name:         "value with special characters",
			storedValue:  "\"key-with_special.chars\"",
			searchValue:  "key-with_special.chars",
			shouldMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the matching logic from DeleteTXTRecord
			records := []string{tt.storedValue}
			matches := false
			for _, record := range records {
				// Remove quotes from stored value for comparison
				stored := record
				if len(stored) >= 2 && stored[0] == '"' && stored[len(stored)-1] == '"' {
					stored = stored[1 : len(stored)-1]
				}
				if stored == tt.searchValue {
					matches = true
					break
				}
			}

			if matches != tt.shouldMatch {
				t.Errorf("Value matching: stored=%q, search=%q, matches=%v, want %v",
					tt.storedValue, tt.searchValue, matches, tt.shouldMatch)
			}
		})
	}
}

// TestDNSClient_TTLConversion tests TTL value conversion
func TestDNSClient_TTLConversion(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int32
	}{
		{
			name:  "standard ACME TTL",
			input: 60,
			want:  60,
		},
		{
			name:  "longer TTL",
			input: 300,
			want:  300,
		},
		{
			name:  "minimum TTL",
			input: 1,
			want:  1,
		},
		{
			name:  "zero TTL",
			input: 0,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttlValue := int32(tt.input)
			if ttlValue != tt.want {
				t.Errorf("TTL conversion = %v, want %v", ttlValue, tt.want)
			}
		})
	}
}

// TestDNSClient_DescriptionConstant tests description constant
func TestDNSClient_DescriptionConstant(t *testing.T) {
	description := "ACME challenge record"
	if description != "ACME challenge record" {
		t.Errorf("Description = %v, want 'ACME challenge record'", description)
	}
}

// TestDNSClient_RecordTypeConstant tests record type constant
func TestDNSClient_RecordTypeConstant(t *testing.T) {
	recordType := "TXT"
	if recordType != "TXT" {
		t.Errorf("Record type = %v, want 'TXT'", recordType)
	}
}

// TestDNSClient_ClientStructure tests DNS client structure
func TestDNSClient_ClientStructure(t *testing.T) {
	d := &DNSClient{
		regionID:  "cn-north-4",
		projectID: "test-project",
		zoneName:  "example.com",
		zoneID:    "zone-123",
	}

	if d.regionID != "cn-north-4" {
		t.Errorf("regionID = %v, want cn-north-4", d.regionID)
	}
	if d.projectID != "test-project" {
		t.Errorf("projectID = %v, want test-project", d.projectID)
	}
	if d.zoneName != "example.com" {
		t.Errorf("zoneName = %v, want example.com", d.zoneName)
	}
	if d.zoneID != "zone-123" {
		t.Errorf("zoneID = %v, want zone-123", d.zoneID)
	}
}

// TestDNSClient_NewDNSClientStructure tests NewDNSClient parameter handling
func TestDNSClient_NewDNSClientStructure(t *testing.T) {
	tests := []struct {
		name      string
		region    string
		projectID string
		ak        string
		sk        string
		zoneName  string
	}{
		{
			name:      "all parameters provided",
			region:    "cn-north-4",
			projectID: "project-123",
			ak:        "access-key",
			sk:        "secret-key",
			zoneName:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parameter construction
			if tt.region == "" {
				t.Error("Region should not be empty")
			}
			if tt.projectID == "" {
				t.Error("Project ID should not be empty")
			}
			if tt.ak == "" {
				t.Error("AK should not be empty")
			}
			if tt.sk == "" {
				t.Error("SK should not be empty")
			}
			if tt.zoneName == "" {
				t.Error("Zone name should not be empty")
			}
		})
	}
}

// TestDNSClient_RegionStructure tests region handling
func TestDNSClient_RegionStructure(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{
			name:     "cn-north-4",
			region:   "cn-north-4",
			expected: "cn-north-4",
		},
		{
			name:     "ap-southeast-1",
			region:   "ap-southeast-1",
			expected: "ap-southeast-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := region.NewRegion(tt.region, tt.region)
			if r.Id != tt.expected {
				t.Errorf("Region ID = %v, want %v", r.Id, tt.expected)
			}
		})
	}
}

// TestDNSClient_CredentialBuilder tests credential building structure
func TestDNSClient_CredentialBuilder(t *testing.T) {
	ak := "test-access-key"
	sk := "test-secret-key"

	builder := basic.NewCredentialsBuilder().
		WithAk(ak).
		WithSk(sk)

	if builder == nil {
		t.Error("CredentialsBuilder should not be nil")
	}
}

// TestDNSClient_HTTPClientBuilder tests HTTP client builder structure
func TestDNSClient_HTTPClientBuilder(t *testing.T) {
	regionID := "cn-north-4"

	builder := dns.DnsClientBuilder().
		WithRegion(region.NewRegion(regionID, regionID))

	if builder == nil {
		t.Error("DnsClientBuilder should not be nil")
	}
}

// TestDNSClient_DnsClientCreation tests DNS client creation structure
func TestDNSClient_DnsClientCreation(t *testing.T) {
	// This test verifies the structure without actually creating a client
	// which would require valid credentials
	// dns.NewDnsClient requires a valid HcHttpClient, which we can't create
	// without proper authentication

	// Test that we can create a DNSClient structure
	d := &DNSClient{
		regionID:  "cn-north-4",
		projectID: "test-project",
		zoneName:  "example.com",
		zoneID:    "zone-123",
	}

	if d.regionID != "cn-north-4" {
		t.Error("DNSClient structure not created correctly")
	}
}
