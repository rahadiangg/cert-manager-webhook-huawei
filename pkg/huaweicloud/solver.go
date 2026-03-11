package huaweicloud

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
)

// extractDNSName extracts user-facing domain from ResolvedFQDN
func extractDNSName(ch *v1alpha1.ChallengeRequest) string {
	// Prefer DNSName for wildcard certificates
	if ch.DNSName != "" && strings.HasPrefix(ch.DNSName, "*.") {
		return strings.TrimPrefix(ch.DNSName, "*.")
	}

	// Extract from ResolvedFQDN
	if ch.ResolvedFQDN != "" {
		fqdn := strings.TrimSuffix(ch.ResolvedFQDN, ".")
		if strings.HasPrefix(fqdn, "_acme-challenge.") {
			return strings.TrimPrefix(fqdn, "_acme-challenge.")
		}
		// Extract zone from FQDN
		parts := strings.Split(fqdn, ".")
		if len(parts) >= 2 {
			return strings.Join(parts[len(parts)-2:], ".")
		}
	}

	return "unknown"
}

// HuaweiCloudSolver implements the webhook.Solver interface for Huawei Cloud DNS
type HuaweiCloudSolver struct {
	client *kubernetes.Clientset
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
func (s *HuaweiCloudSolver) Name() string {
	return "huawei-solver"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (s *HuaweiCloudSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		Error("ACME challenge: failed to load config",
			"uid", ch.UID,
			"action", "present",
			"error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	dnsName := extractDNSName(ch)

	Info("ACME challenge: Present started",
		"uid", ch.UID,
		"action", "present",
		"dns_name", dnsName,
		"resolved_fqdn", ch.ResolvedFQDN,
		"namespace", ch.ResourceNamespace,
	)

	// Get credentials from Kubernetes Secret
	ak, sk, err := s.getCredentials(ch.ResourceNamespace, cfg.AKSecretRef, cfg.SKSecretRef)
	if err != nil {
		Error("ACME challenge: failed to get credentials",
			"uid", ch.UID,
			"action", "present",
			"dns_name", dnsName,
			"namespace", ch.ResourceNamespace,
			"error", err)
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Create DNS client
	dnsClient, err := NewDNSClient(cfg.Region, cfg.ProjectID, ak, sk, cfg.ZoneName)
	if err != nil {
		Error("ACME challenge: failed to create DNS client",
			"uid", ch.UID,
			"action", "present",
			"dns_name", dnsName,
			"region", cfg.Region,
			"project_id", cfg.ProjectID,
			"error", err)
		return fmt.Errorf("failed to create DNS client: %w", err)
	}

	// Create operation context for DNS operations
	ctx := OperationContext{
		UID:     string(ch.UID),
		Action:  "present",
		DNSName: dnsName,
	}

	// Create TXT record
	// Use a reasonable TTL - 60 seconds is typical for ACME challenges
	ttl := 60
	err = dnsClient.CreateTXTRecord(ctx, ch.ResolvedFQDN, ch.Key, ttl)
	if err != nil {
		Error("ACME challenge: failed to create TXT record",
			"uid", ch.UID,
			"action", "present",
			"dns_name", dnsName,
			"record_name", ch.ResolvedFQDN,
			"ttl", ttl,
			"error", err)
		return fmt.Errorf("failed to create TXT record: %w", err)
	}

	Info("ACME challenge: Present completed",
		"uid", ch.UID,
		"action", "present",
		"dns_name", dnsName,
		"zone", cfg.ZoneName,
		"record_name", ch.ResolvedFQDN,
	)
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (s *HuaweiCloudSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		Error("ACME challenge: failed to load config",
			"uid", ch.UID,
			"action", "cleanup",
			"error", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	dnsName := extractDNSName(ch)

	Info("ACME challenge: CleanUp started",
		"uid", ch.UID,
		"action", "cleanup",
		"dns_name", dnsName,
		"resolved_fqdn", ch.ResolvedFQDN,
	)

	// Get credentials from Kubernetes Secret
	ak, sk, err := s.getCredentials(ch.ResourceNamespace, cfg.AKSecretRef, cfg.SKSecretRef)
	if err != nil {
		Error("ACME challenge: failed to get credentials",
			"uid", ch.UID,
			"action", "cleanup",
			"dns_name", dnsName,
			"namespace", ch.ResourceNamespace,
			"error", err)
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Create DNS client
	dnsClient, err := NewDNSClient(cfg.Region, cfg.ProjectID, ak, sk, cfg.ZoneName)
	if err != nil {
		Error("ACME challenge: failed to create DNS client",
			"uid", ch.UID,
			"action", "cleanup",
			"dns_name", dnsName,
			"region", cfg.Region,
			"project_id", cfg.ProjectID,
			"error", err)
		return fmt.Errorf("failed to create DNS client: %w", err)
	}

	// Create operation context for DNS operations
	ctx := OperationContext{
		UID:     string(ch.UID),
		Action:  "cleanup",
		DNSName: dnsName,
	}

	// Delete the specific TXT record matching the key
	err = dnsClient.DeleteTXTRecord(ctx, ch.ResolvedFQDN, ch.Key)
	if err != nil {
		Error("ACME challenge: failed to delete TXT record",
			"uid", ch.UID,
			"action", "cleanup",
			"dns_name", dnsName,
			"record_name", ch.ResolvedFQDN,
			"error", err)
		return fmt.Errorf("failed to delete TXT record: %w", err)
	}

	Info("ACME challenge: CleanUp completed",
		"uid", ch.UID,
		"action", "cleanup",
		"dns_name", dnsName,
		"zone", cfg.ZoneName,
	)
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (s *HuaweiCloudSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		Error("failed to create kubernetes clientset", "error", err)
		return fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	s.client = cl

	Info("HuaweiCloudSolver initialized successfully",
		"solver_name", s.Name(),
	)
	return nil
}

// getCredentials retrieves the AK and SK from Kubernetes Secrets
func (s *HuaweiCloudSolver) getCredentials(namespace string, akRef, skRef SecretKeySelector) (string, string, error) {
	if s.client == nil {
		return "", "", fmt.Errorf("kubernetes client not initialized")
	}

	// Get AK Secret
	akSecret, err := s.client.CoreV1().Secrets(namespace).Get(context.TODO(), akRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get AK secret %s/%s: %w", namespace, akRef.Name, err)
	}

	akBytes, ok := akSecret.Data[akRef.Key]
	if !ok {
		return "", "", fmt.Errorf("key %s not found in secret %s/%s", akRef.Key, namespace, akRef.Name)
	}

	// Get SK Secret (may be from same or different secret)
	skSecret, err := s.client.CoreV1().Secrets(namespace).Get(context.TODO(), skRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to get SK secret %s/%s: %w", namespace, skRef.Name, err)
	}

	skBytes, ok := skSecret.Data[skRef.Key]
	if !ok {
		return "", "", fmt.Errorf("key %s not found in secret %s/%s", skRef.Key, namespace, skRef.Name)
	}

	return string(akBytes), string(skBytes), nil
}
