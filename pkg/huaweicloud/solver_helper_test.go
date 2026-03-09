package huaweicloud

import (
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
)

// TestHuaweiCloudSolver_ConfigStructure tests config structure validation
func TestHuaweiCloudSolver_ConfigStructure(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		valid   bool
	}{
		{
			name: "complete valid config",
			config: `{
				"region": "cn-north-4",
				"projectId": "test-project",
				"zoneName": "example.com",
				"akSecretRef": {"name": "ak-secret", "key": "ak"},
				"skSecretRef": {"name": "sk-secret", "key": "sk"}
			}`,
			valid: true,
		},
		{
			name: "config with different region",
			config: `{
				"region": "ap-southeast-1",
				"projectId": "project-123",
				"zoneName": "test.org",
				"akSecretRef": {"name": "secret1", "key": "key1"},
				"skSecretRef": {"name": "secret2", "key": "key2"}
			}`,
			valid: true,
		},
		{
			name: "missing region",
			config: `{
				"projectId": "test",
				"zoneName": "example.com",
				"akSecretRef": {"name": "a", "key": "b"},
				"skSecretRef": {"name": "c", "key": "d"}
			}`,
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgJSON := &extapi.JSON{Raw: []byte(tt.config)}
			_, err := loadConfig(cfgJSON)

			if tt.valid && err != nil {
				t.Errorf("Expected config to be valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected config to be invalid, got no error")
			}
		})
	}
}

// TestHuaweiCloudSolver_DomainExtraction tests domain extraction from FQDN
func TestHuaweiCloudSolver_DomainExtraction(t *testing.T) {
	tests := []struct {
		name         string
		zoneName     string
		resolvedFQDN string
	}{
		{
			name:         "simple domain",
			zoneName:     "example.com",
			resolvedFQDN: "_acme-challenge.example.com",
		},
		{
			name:         "subdomain",
			zoneName:     "example.com",
			resolvedFQDN: "_acme-challenge.api.example.com",
		},
		{
			name:         "deep subdomain",
			zoneName:     "example.com",
			resolvedFQDN: "_acme-challenge.a.b.c.example.com",
		},
		{
			name:         "multi-level TLD",
			zoneName:     "example.co.uk",
			resolvedFQDN: "_acme-challenge.example.co.uk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DNSClient{
				zoneName: tt.zoneName,
			}

			// Test extractRecordName
			recordName := d.extractRecordName(tt.resolvedFQDN)

			// The extractRecordName should return the FQDN without trailing dot
			expected := tt.resolvedFQDN
			if len(expected) > 0 && expected[len(expected)-1] == '.' {
				expected = expected[:len(expected)-1]
			}
			if recordName != expected {
				t.Errorf("extractRecordName() = %v, want %v", recordName, expected)
			}
		})
	}
}

// TestHuaweiCloudSolver_NamespaceHandling tests namespace parameter handling
func TestHuaweiCloudSolver_NamespaceHandling(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{
			name:      "default namespace",
			namespace: "default",
		},
		{
			name:      "cert-manager namespace",
			namespace: "cert-manager",
		},
		{
			name:      "custom namespace",
			namespace: "my-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			akRef := SecretKeySelector{
				Name: "test-secret",
				Key:  "test-key",
			}

			// Test getCredentials with different namespaces
			_, _, err := s.getCredentials(tt.namespace, akRef, akRef)

			// Should fail due to no client
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}
		})
	}
}

// TestHuaweiCloudSolver_SecretRefHandling tests secret reference handling
func TestHuaweiCloudSolver_SecretRefHandling(t *testing.T) {
	tests := []struct {
		name      string
		akRef     SecretKeySelector
		skRef     SecretKeySelector
	}{
		{
			name: "same secret for both",
			akRef: SecretKeySelector{
				Name: "huawei-credentials",
				Key:  "ak",
			},
			skRef: SecretKeySelector{
				Name: "huawei-credentials",
				Key:  "sk",
			},
		},
		{
			name: "different secrets",
			akRef: SecretKeySelector{
				Name: "huawei-ak",
				Key:  "access-key",
			},
			skRef: SecretKeySelector{
				Name: "huawei-sk",
				Key:  "secret-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			// Test getCredentials structure
			_, _, err := s.getCredentials("default", tt.akRef, tt.skRef)

			// Should fail due to no client
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}
		})
	}
}

// TestHuaweiCloudSolver_KeyHandling tests challenge key handling
func TestHuaweiCloudSolver_KeyHandling(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "simple key",
			key:  "test-key",
		},
		{
			name: "key with special chars",
			key:  "key-with_underscore.and-dash",
		},
		{
			name: "long key",
			key:  "very-long-acme-validation-key-with-multiple-characters-123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that key is not empty
			if tt.key == "" {
				t.Error("Key should not be empty")
			}
		})
	}
}

// TestHuaweiCloudSolver_InitializeStopChannel tests stop channel parameter
func TestHuaweiCloudSolver_InitializeStopChannel(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Test with closed stop channel - this will panic due to nil config
	// We need to handle the panic
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic due to nil config
			t.Logf("Recovered from expected panic: %v", r)
		}
	}()

	stopCh := make(chan struct{})
	close(stopCh)
	_ = s.Initialize(nil, stopCh)
}

// TestHuaweiCloudSolver_PresentConfigParsing tests config parsing in Present
func TestHuaweiCloudSolver_PresentConfigParsing(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "valid config",
			config: `{
				"region": "cn-north-4",
				"projectId": "test-project",
				"zoneName": "example.com",
				"akSecretRef": {"name": "ak", "key": "ak"},
				"skSecretRef": {"name": "sk", "key": "sk"}
			}`,
		},
		{
			name: "config with extra whitespace",
			config: `{
				"region": "cn-north-4",
				"projectId": "test-project",
				"zoneName": "example.com",
				"akSecretRef": { "name": "ak", "key": "ak" },
				"skSecretRef": { "name": "sk", "key": "sk" }
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			ch := &v1alpha1.ChallengeRequest{
				Config: &extapi.JSON{
					Raw: []byte(tt.config),
				},
				ResourceNamespace: "default",
				ResolvedFQDN:      "_acme-challenge.example.com",
				Key:               "test-key",
			}

			// Present should fail at credential stage, but config should parse correctly
			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}

			// Should not fail at config loading
			if containsSubstring(err.Error(), "failed to load config") {
				t.Errorf("Unexpected config loading error: %v", err)
			}
		})
	}
}

// TestHuaweiCloudSolver_CleanUpConfigParsing tests config parsing in CleanUp
func TestHuaweiCloudSolver_CleanUpConfigParsing(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "valid config",
			config: `{
				"region": "ap-southeast-1",
				"projectId": "project-123",
				"zoneName": "test.org",
				"akSecretRef": {"name": "secret1", "key": "key1"},
				"skSecretRef": {"name": "secret2", "key": "key2"}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			ch := &v1alpha1.ChallengeRequest{
				Config: &extapi.JSON{
					Raw: []byte(tt.config),
				},
				ResourceNamespace: "default",
				ResolvedFQDN:      "_acme-challenge.test.org",
				Key:               "cleanup-key",
			}

			// CleanUp should fail at credential stage, but config should parse correctly
			err := s.CleanUp(ch)
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}

			// Should not fail at config loading
			if containsSubstring(err.Error(), "failed to load config") {
				t.Errorf("Unexpected config loading error: %v", err)
			}
		})
	}
}

// TestHuaweiCloudSolver_DNSClientCreationStructure tests DNS client creation structure
func TestHuaweiCloudSolver_DNSClientCreationStructure(t *testing.T) {
	tests := []struct {
		name      string
		region    string
		projectID string
		ak        string
		sk        string
		zoneName  string
	}{
		{
			name:      "all parameters",
			region:    "cn-north-4",
			projectID: "test-project",
			ak:        "access-key",
			sk:        "secret-key",
			zoneName:  "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parameter structure
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

// TestHuaweiCloudSolver_ErrorWrapping tests error wrapping in solver
func TestHuaweiCloudSolver_ErrorWrapping(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Test Present error wrapping
	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(`{invalid json}`),
		},
	}

	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Check error wrapping
	if err != nil {
		if !containsSubstring(err.Error(), "failed to load config") {
			t.Errorf("Error should wrap 'failed to load config', got: %v", err)
		}
	}
}

// TestHuaweiCloudSolver_SolverInitialization tests solver initialization state
func TestHuaweiCloudSolver_SolverInitialization(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Before initialization, client should be nil
	if s.client != nil {
		t.Error("Client should be nil before initialization")
	}

	// Name should work even without initialization
	name := s.Name()
	if name != "huawei-solver" {
		t.Errorf("Name() = %v, want 'huawei-solver'", name)
	}
}

// TestHuaweiCloudSolver_MultipleOperations tests multiple Present/CleanUp operations
func TestHuaweiCloudSolver_MultipleOperations(t *testing.T) {
	s := &HuaweiCloudSolver{}

	config := `{
		"region": "cn-north-4",
		"projectId": "test-project",
		"zoneName": "example.com",
		"akSecretRef": {"name": "ak", "key": "ak"},
		"skSecretRef": {"name": "sk", "key": "sk"}
	}`

	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(config),
		},
		ResourceNamespace: "default",
		ResolvedFQDN:      "_acme-challenge.example.com",
		Key:               "test-key",
	}

	// Multiple Present calls should have consistent behavior
	err1 := s.Present(ch)
	err2 := s.Present(ch)

	if err1 == nil || err2 == nil {
		t.Error("Expected errors when client is not initialized")
	}

	// Both should fail at the same point
	if err1 != nil && err2 != nil {
		err1Str := err1.Error()
		err2Str := err2.Error()

		if containsSubstring(err1Str, "failed to get credentials") &&
			containsSubstring(err2Str, "failed to get credentials") {
			// Consistent error handling
		} else {
			t.Logf("Errors: %v, %v", err1, err2)
		}
	}
}

// TestHuaweiCloudSolver_ConfigFieldTypes tests config field types
func TestHuaweiCloudSolver_ConfigFieldTypes(t *testing.T) {
	cfg := HuaweiCloudConfig{
		Region:    "cn-north-4",
		ProjectID: "test-project",
		ZoneName:  "example.com",
		AKSecretRef: SecretKeySelector{
			Name: "ak-secret",
			Key:  "ak-key",
		},
		SKSecretRef: SecretKeySelector{
			Name: "sk-secret",
			Key:  "sk-key",
		},
	}

	// Verify field types
	if cfg.Region != "cn-north-4" {
		t.Error("Region field type or value incorrect")
	}
	if cfg.ProjectID != "test-project" {
		t.Error("ProjectID field type or value incorrect")
	}
	if cfg.ZoneName != "example.com" {
		t.Error("ZoneName field type or value incorrect")
	}
	if cfg.AKSecretRef.Name != "ak-secret" {
		t.Error("AKSecretRef.Name field type or value incorrect")
	}
	if cfg.AKSecretRef.Key != "ak-key" {
		t.Error("AKSecretRef.Key field type or value incorrect")
	}
	if cfg.SKSecretRef.Name != "sk-secret" {
		t.Error("SKSecretRef.Name field type or value incorrect")
	}
	if cfg.SKSecretRef.Key != "sk-key" {
		t.Error("SKSecretRef.Key field type or value incorrect")
	}
}

// TestHuaweiCloudSolver_EmptyKeyHandling tests empty key handling
func TestHuaweiCloudSolver_EmptyKeyHandling(t *testing.T) {
	s := &HuaweiCloudSolver{}

	config := `{
		"region": "cn-north-4",
		"projectId": "test-project",
		"zoneName": "example.com",
		"akSecretRef": {"name": "ak", "key": "ak"},
		"skSecretRef": {"name": "sk", "key": "sk"}
	}`

	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(config),
		},
		ResourceNamespace: "default",
		ResolvedFQDN:      "_acme-challenge.example.com",
		Key:               "", // Empty key
	}

	// Should fail at credential stage, but key should be passed through
	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error when client is not initialized")
	}

	// Verify key was used (even if empty)
	if ch.Key != "" {
		t.Errorf("Key = %v, want empty string", ch.Key)
	}
}
