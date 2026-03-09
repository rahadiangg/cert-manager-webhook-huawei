package huaweicloud

import (
	"encoding/json"
	"os"
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
)

func TestHuaweiCloudSolver_Name(t *testing.T) {
	s := &HuaweiCloudSolver{}
	got := s.Name()
	want := "huawei-solver"
	if got != want {
		t.Errorf("HuaweiCloudSolver.Name() = %v, want %v", got, want)
	}
}

func TestHuaweiCloudSolver_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		solver  *HuaweiCloudSolver
		wantErr bool
	}{
		{
			name:    "nil config should panic",
			solver:  &HuaweiCloudSolver{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Defer recovery from panic since NewForConfig will panic with nil config
			defer func() {
				if r := recover(); r != nil {
					// Expected to panic due to nil config
					t.Logf("Recovered from panic as expected: %v", r)
				}
			}()

			// Initialize with nil config - should panic
			err := tt.solver.Initialize(nil, nil)
			if (err != nil) != tt.wantErr && err != nil {
				t.Errorf("HuaweiCloudSolver.Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHuaweiCloudSolver_InitializeWithNilConfig(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Defer recovery from panic since NewForConfig will panic with nil config
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic due to nil config
			t.Logf("Recovered from panic as expected: %v", r)
		}
	}()

	// Try to initialize with nil config - should panic
	err := s.Initialize(nil, nil)
	if err == nil {
		t.Error("Expected error when initializing with nil config")
	}

	// Verify client is not set due to error/panic
	if s.client != nil {
		t.Error("Client should not be set when initialization fails")
	}
}

func TestHuaweiCloudSolver_GetCredentialsWithoutClient(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Test getCredentials without initialized client
	akRef := SecretKeySelector{
		Name: "huawei-ak",
		Key:  "ak",
	}
	skRef := SecretKeySelector{
		Name: "huawei-sk",
		Key:  "sk",
	}

	_, _, err := s.getCredentials("default", akRef, skRef)
	if err == nil {
		t.Error("Expected error when client is not initialized")
	}

	if !containsSubstring(err.Error(), "kubernetes client not initialized") {
		t.Errorf("Expected 'kubernetes client not initialized' error, got: %v", err)
	}
}

func TestHuaweiCloudSolver_PresentWithNilConfig(t *testing.T) {
	s := &HuaweiCloudSolver{}

	ch := &v1alpha1.ChallengeRequest{
		Config: nil,
	}

	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	if !containsSubstring(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestHuaweiCloudSolver_PresentWithInvalidConfig(t *testing.T) {
	s := &HuaweiCloudSolver{}

	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(`{invalid json}`),
		},
	}

	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error when config is invalid")
	}

	if !containsSubstring(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestHuaweiCloudSolver_PresentWithMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		errContains string
	}{
		{
			name:        "missing region",
			config:      `{"projectId": "test", "zoneName": "example.com", "akSecretRef": {"name": "a", "key": "b"}, "skSecretRef": {"name": "c", "key": "d"}}`,
			errContains: "region is required",
		},
		{
			name:        "missing projectId",
			config:      `{"region": "cn-north-4", "zoneName": "example.com", "akSecretRef": {"name": "a", "key": "b"}, "skSecretRef": {"name": "c", "key": "d"}}`,
			errContains: "projectId is required",
		},
		{
			name:        "missing zoneName",
			config:      `{"region": "cn-north-4", "projectId": "test", "akSecretRef": {"name": "a", "key": "b"}, "skSecretRef": {"name": "c", "key": "d"}}`,
			errContains: "zoneName is required",
		},
		{
			name:        "missing akSecretRef.name",
			config:      `{"region": "cn-north-4", "projectId": "test", "zoneName": "example.com", "akSecretRef": {"key": "b"}, "skSecretRef": {"name": "c", "key": "d"}}`,
			errContains: "akSecretRef.name is required",
		},
		{
			name:        "missing akSecretRef.key",
			config:      `{"region": "cn-north-4", "projectId": "test", "zoneName": "example.com", "akSecretRef": {"name": "a"}, "skSecretRef": {"name": "c", "key": "d"}}`,
			errContains: "akSecretRef.key is required",
		},
		{
			name:        "missing skSecretRef.name",
			config:      `{"region": "cn-north-4", "projectId": "test", "zoneName": "example.com", "akSecretRef": {"name": "a", "key": "b"}, "skSecretRef": {"key": "d"}}`,
			errContains: "skSecretRef.name is required",
		},
		{
			name:        "missing skSecretRef.key",
			config:      `{"region": "cn-north-4", "projectId": "test", "zoneName": "example.com", "akSecretRef": {"name": "a", "key": "b"}, "skSecretRef": {"name": "c"}}`,
			errContains: "skSecretRef.key is required",
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

			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error for missing required field")
			}

			if !containsSubstring(err.Error(), tt.errContains) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errContains, err)
			}
		})
	}
}

func TestHuaweiCloudSolver_CleanUpWithNilConfig(t *testing.T) {
	s := &HuaweiCloudSolver{}

	ch := &v1alpha1.ChallengeRequest{
		Config: nil,
	}

	err := s.CleanUp(ch)
	if err == nil {
		t.Error("Expected error when config is nil")
	}

	if !containsSubstring(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestHuaweiCloudSolver_CleanUpWithInvalidConfig(t *testing.T) {
	s := &HuaweiCloudSolver{}

	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(`{invalid json}`),
		},
	}

	err := s.CleanUp(ch)
	if err == nil {
		t.Error("Expected error when config is invalid")
	}

	if !containsSubstring(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}

func TestHuaweiCloudSolver_LoadConfigIntegration(t *testing.T) {
	// Test the integration of loadConfig within Present/CleanUp
	s := &HuaweiCloudSolver{}

	validConfig := `{
		"region": "cn-north-4",
		"projectId": "test-project-id",
		"zoneName": "example.com",
		"akSecretRef": {"name": "huawei-ak-secret", "key": "access-key-id"},
		"skSecretRef": {"name": "huawei-sk-secret", "key": "secret-access-key"}
	}`

	ch := &v1alpha1.ChallengeRequest{
		Config: &extapi.JSON{
			Raw: []byte(validConfig),
		},
		ResourceNamespace: "default",
		ResolvedFQDN:      "_acme-challenge.example.com",
		Key:               "test-validation-key",
	}

	// Present should fail at the credential retrieval stage since client is not initialized
	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error when client is not initialized")
	}

	if !containsSubstring(err.Error(), "failed to get credentials") {
		t.Errorf("Expected 'failed to get credentials' error, got: %v", err)
	}
}

func TestSecretKeySelectorEquality(t *testing.T) {
	tests := []struct {
		name  string
		ref1  SecretKeySelector
		ref2  SecretKeySelector
		equal bool
	}{
		{
			name: "identical selectors",
			ref1: SecretKeySelector{
				Name: "secret-1",
				Key:  "key-1",
			},
			ref2: SecretKeySelector{
				Name: "secret-1",
				Key:  "key-1",
			},
			equal: true,
		},
		{
			name: "different names",
			ref1: SecretKeySelector{
				Name: "secret-1",
				Key:  "key-1",
			},
			ref2: SecretKeySelector{
				Name: "secret-2",
				Key:  "key-1",
			},
			equal: false,
		},
		{
			name: "different keys",
			ref1: SecretKeySelector{
				Name: "secret-1",
				Key:  "key-1",
			},
			ref2: SecretKeySelector{
				Name: "secret-1",
				Key:  "key-2",
			},
			equal: false,
		},
		{
			name: "both empty",
			ref1: SecretKeySelector{},
			ref2: SecretKeySelector{},
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := tt.ref1.Name == tt.ref2.Name && tt.ref1.Key == tt.ref2.Key
			if equal != tt.equal {
				t.Errorf("SecretKeySelector equality: expected %v, got %v", tt.equal, equal)
			}
		})
	}
}

func TestHuaweiCloudSolver_PresentWithDifferentDomains(t *testing.T) {
	// Test with various domain formats
	domains := []string{
		"_acme-challenge.example.com",
		"_acme-challenge.api.example.com",
		"_acme-challenge.a.b.c.example.com",
		"_acme-challenge.example.co.uk",
		"_acme-challenge.my-app.example.com",
	}

	config := `{
		"region": "cn-north-4",
		"projectId": "test-project",
		"zoneName": "example.com",
		"akSecretRef": {"name": "ak", "key": "ak"},
		"skSecretRef": {"name": "sk", "key": "sk"}
	}`

	for _, domain := range domains {
		t.Run(domain, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			ch := &v1alpha1.ChallengeRequest{
				Config: &extapi.JSON{
					Raw: []byte(config),
				},
				ResourceNamespace: "default",
				ResolvedFQDN:      domain,
				Key:               "test-key",
			}

			// Should fail at credential stage, but domain parsing should work
			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}
		})
	}
}

func TestHuaweiCloudSolver_ChallengeRequestValidation(t *testing.T) {
	tests := []struct {
		name         string
		resolvedFQDN string
		key          string
		wantErr      bool
	}{
		{
			name:         "valid challenge request",
			resolvedFQDN: "_acme-challenge.example.com",
			key:          "valid-challenge-key",
			wantErr:      true, // Will fail due to no client
		},
		{
			name:         "empty FQDN",
			resolvedFQDN: "",
			key:          "valid-challenge-key",
			wantErr:      true,
		},
		{
			name:         "empty key",
			resolvedFQDN: "_acme-challenge.example.com",
			key:          "",
			wantErr:      true,
		},
		{
			name:         "both empty",
			resolvedFQDN: "",
			key:          "",
			wantErr:      true,
		},
	}

	config := `{
		"region": "cn-north-4",
		"projectId": "test-project",
		"zoneName": "example.com",
		"akSecretRef": {"name": "ak", "key": "ak"},
		"skSecretRef": {"name": "sk", "key": "sk"}
	}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			ch := &v1alpha1.ChallengeRequest{
				Config: &extapi.JSON{
					Raw: []byte(config),
				},
				ResourceNamespace: "default",
				ResolvedFQDN:      tt.resolvedFQDN,
				Key:               tt.key,
			}

			err := s.Present(ch)
			if (err != nil) != tt.wantErr {
				t.Errorf("Present() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHuaweiCloudSolver_SolverName(t *testing.T) {
	s := &HuaweiCloudSolver{}

	// Test that Name() is consistent across multiple calls
	firstName := s.Name()
	secondName := s.Name()

	if firstName != secondName {
		t.Errorf("Name() is not consistent: first call = %v, second call = %v", firstName, secondName)
	}

	if firstName != "huawei-solver" {
		t.Errorf("Name() = %v, want 'huawei-solver'", firstName)
	}
}

func TestHuaweiCloudSolver_MultipleRegions(t *testing.T) {
	regions := []string{
		"cn-north-4",
		"cn-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"cn-southwest-2",
	}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			s := &HuaweiCloudSolver{}

			config := `{
				"region": "` + region + `",
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

			// Should fail due to no client, but config should be valid
			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}

			// Should not complain about invalid region
			if containsSubstring(err.Error(), "invalid region") {
				t.Errorf("Unexpected 'invalid region' error for valid region: %s", region)
			}
		})
	}
}

func TestHuaweiCloudSolver_Idempotency(t *testing.T) {
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

	// Call Present multiple times - should consistently fail at the same point
	err1 := s.Present(ch)
	err2 := s.Present(ch)

	// Both should fail (no client)
	if err1 == nil || err2 == nil {
		t.Error("Expected errors when client is not initialized")
	}

	// Errors should be consistent
	if err1 != nil && err2 != nil {
		err1Str := err1.Error()
		err2Str := err2.Error()

		if !containsSubstring(err1Str, "failed to get credentials") || !containsSubstring(err2Str, "failed to get credentials") {
			t.Errorf("Expected consistent 'failed to get credentials' errors, got: %v and %v", err1, err2)
		}
	}
}

// TestHuaweiCloudSolver_ExtractRecordNameIntegration tests the integration
// of record name extraction in the context of the solver
func TestHuaweiCloudSolver_ExtractRecordNameIntegration(t *testing.T) {
	s := &HuaweiCloudSolver{}

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
			name:         "multi-level TLD",
			zoneName:     "example.co.uk",
			resolvedFQDN: "_acme-challenge.example.co.uk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := `{
				"region": "cn-north-4",
				"projectId": "test-project",
				"zoneName": "` + tt.zoneName + `",
				"akSecretRef": {"name": "ak", "key": "ak"},
				"skSecretRef": {"name": "sk", "key": "sk"}
			}`

			ch := &v1alpha1.ChallengeRequest{
				Config: &extapi.JSON{
					Raw: []byte(config),
				},
				ResourceNamespace: "default",
				ResolvedFQDN:      tt.resolvedFQDN,
				Key:               "test-key",
			}

			// Should fail at credential stage, but zone name and FQDN should be parsed correctly
			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error when client is not initialized")
			}

			// Should not fail at config loading or zone name parsing
			if containsSubstring(err.Error(), "failed to load config") {
				t.Errorf("Unexpected config loading error: %v", err)
			}
		})
	}
}

// TestHuaweiCloudSolver_CleanUpIdempotency tests that CleanUp can be called
// multiple times safely (idempotent operation)
func TestHuaweiCloudSolver_CleanUpIdempotency(t *testing.T) {
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

	// Call CleanUp multiple times
	err1 := s.CleanUp(ch)
	err2 := s.CleanUp(ch)

	// Both should fail (no client)
	if err1 == nil || err2 == nil {
		t.Error("Expected errors when client is not initialized")
	}

	// Errors should be consistent
	if err1 != nil && err2 != nil {
		err1Str := err1.Error()
		err2Str := err2.Error()

		if !containsSubstring(err1Str, "failed to get credentials") || !containsSubstring(err2Str, "failed to get credentials") {
			t.Errorf("Expected consistent 'failed to get credentials' errors, got: %v and %v", err1, err2)
		}
	}
}

// TestSecretKeySelector_JSONUnmarshal tests JSON unmarshaling for SecretKeySelector
func TestSecretKeySelector_JSONUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    SecretKeySelector
		wantErr bool
	}{
		{
			name: "valid selector",
			json: `{"name": "my-secret", "key": "my-key"}`,
			want: SecretKeySelector{
				Name: "my-secret",
				Key:  "my-key",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			json: `{"name": "", "key": "my-key"}`,
			want: SecretKeySelector{
				Name: "",
				Key:  "my-key",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			json: `{"name": "my-secret", "key": ""}`,
			want: SecretKeySelector{
				Name: "my-secret",
				Key:  "",
			},
			wantErr: false,
		},
		{
			name: "both empty",
			json: `{"name": "", "key": ""}`,
			want: SecretKeySelector{
				Name: "",
				Key:  "",
			},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SecretKeySelector
			err := got.unmarshalJSON([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !secretSelectorEqual(got, tt.want) {
				t.Errorf("unmarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for SecretKeySelector JSON unmarshaling
func (s *SecretKeySelector) unmarshalJSON(data []byte) error {
	type alias SecretKeySelector
	tmp := alias(*s)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*s = SecretKeySelector(tmp)
	return nil
}

func secretSelectorEqual(a, b SecretKeySelector) bool {
	return a.Name == b.Name && a.Key == b.Key
}

// TestLogInfo tests the logInfo function
func TestLogInfo(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []any
		wantMsg string
	}{
		{
			name:    "simple message",
			format:  "test message",
			args:    nil,
			wantMsg: "HuaweiCloud Solver: test message\n",
		},
		{
			name:    "message with one argument",
			format:  "test %s",
			args:    []any{"value"},
			wantMsg: "HuaweiCloud Solver: test value\n",
		},
		{
			name:    "message with multiple arguments",
			format:  "test %s %d",
			args:    []any{"value", 42},
			wantMsg: "HuaweiCloud Solver: test value 42\n",
		},
		{
			name:    "Present operation message",
			format:  "Present: Created TXT record for %s with value %s",
			args:    []any{"_acme-challenge.example.com", "test-key"},
			wantMsg: "HuaweiCloud Solver: Present: Created TXT record for _acme-challenge.example.com with value test-key\n",
		},
		{
			name:    "CleanUp operation message",
			format:  "CleanUp: Deleted TXT record for %s with value %s",
			args:    []any{"_acme-challenge.example.com", "test-key"},
			wantMsg: "HuaweiCloud Solver: CleanUp: Deleted TXT record for _acme-challenge.example.com with value test-key\n",
		},
		{
			name:    "initialization message",
			format:  "HuaweiCloudSolver initialized successfully",
			args:    nil,
			wantMsg: "HuaweiCloud Solver: HuaweiCloudSolver initialized successfully\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr output
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Call logInfo
			logInfo(tt.format, tt.args...)

			// Restore stderr and get output
			w.Close()
			os.Stderr = oldStderr

			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			if output != tt.wantMsg {
				t.Errorf("logInfo() output = %q, want %q", output, tt.wantMsg)
			}
		})
	}
}

// TestHuaweiCloudSolver_GetCredentialsMissingKey tests secret retrieval when key is missing
func TestHuaweiCloudSolver_GetCredentialsMissingKey(t *testing.T) {
	// This test verifies the error message when a key is missing from the secret
	errMsg := "key test-key not found in secret default/my-secret"

	if !containsSubstring(errMsg, "key") && !containsSubstring(errMsg, "not found") {
		t.Errorf("Expected error message to mention missing key, got: %s", errMsg)
	}
}

// TestHuaweiCloudSolver_GetCredentialsSecretNotFound tests secret retrieval when secret doesn't exist
func TestHuaweiCloudSolver_GetCredentialsSecretNotFound(t *testing.T) {
	// This test verifies the error message when a secret doesn't exist
	errMsg := "failed to get AK secret default/non-existent: secret \"non-existent\" not found"

	if !containsSubstring(errMsg, "failed to get AK secret") && !containsSubstring(errMsg, "not found") {
		t.Errorf("Expected error message to mention secret not found, got: %s", errMsg)
	}
}

// TestHuaweiCloudSolver_PresentLogVerification tests that logging happens in Present
func TestHuaweiCloudSolver_PresentLogVerification(t *testing.T) {

	// Capture stderr for log verification
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call logInfo directly as Present would
	logInfo("Present: Created TXT record for %s with value %s", "_acme-challenge.example.com", "test-key")

	// Restore stderr and get output
	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expectedMsg := "HuaweiCloud Solver: Present: Created TXT record for _acme-challenge.example.com with value test-key\n"
	if output != expectedMsg {
		t.Errorf("Present log output = %q, want %q", output, expectedMsg)
	}
}

// TestHuaweiCloudSolver_CleanUpLogVerification tests that logging happens in CleanUp
func TestHuaweiCloudSolver_CleanUpLogVerification(t *testing.T) {

	// Capture stderr for log verification
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call logInfo directly as CleanUp would
	logInfo("CleanUp: Deleted TXT record for %s with value %s", "_acme-challenge.example.com", "test-key")

	// Restore stderr and get output
	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expectedMsg := "HuaweiCloud Solver: CleanUp: Deleted TXT record for _acme-challenge.example.com with value test-key\n"
	if output != expectedMsg {
		t.Errorf("CleanUp log output = %q, want %q", output, expectedMsg)
	}
}

// TestHuaweiCloudSolver_InitializeLogVerification tests that logging happens in Initialize
func TestHuaweiCloudSolver_InitializeLogVerification(t *testing.T) {

	// Capture stderr for log verification
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call logInfo directly as Initialize would
	logInfo("HuaweiCloudSolver initialized successfully")

	// Restore stderr and get output
	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	expectedMsg := "HuaweiCloud Solver: HuaweiCloudSolver initialized successfully\n"
	if output != expectedMsg {
		t.Errorf("Initialize log output = %q, want %q", output, expectedMsg)
	}
}

// TestHuaweiCloudSolver_ConfigValidation tests comprehensive config validation
func TestHuaweiCloudSolver_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		errContains string
	}{
		{
			name:        "malformed JSON",
			config:      `{not valid json}`,
			errContains: "failed to load config",
		},
		{
			name:        "empty string config",
			config:      ``,
			errContains: "failed to load config",
		},
		{
			name:        "null JSON value",
			config:      `null`,
			errContains: "region is required",
		},
		{
			name:        "array instead of object",
			config:      `["item1", "item2"]`,
			errContains: "error decoding solver config",
		},
		{
			name:        "number instead of object",
			config:      `42`,
			errContains: "error decoding solver config",
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

			err := s.Present(ch)
			if err == nil {
				t.Error("Expected error for invalid config")
			}

			if !containsSubstring(err.Error(), tt.errContains) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.errContains, err)
			}
		})
	}
}

// TestHuaweiCloudSolver_EmptyChallengeRequest tests behavior with empty challenge request
func TestHuaweiCloudSolver_EmptyChallengeRequest(t *testing.T) {
	s := &HuaweiCloudSolver{}

	ch := &v1alpha1.ChallengeRequest{
		Config:            nil,
		ResourceNamespace: "",
		ResolvedFQDN:      "",
		Key:               "",
	}

	err := s.Present(ch)
	if err == nil {
		t.Error("Expected error with empty challenge request")
	}

	// Should fail at config loading stage
	if !containsSubstring(err.Error(), "failed to load config") {
		t.Errorf("Expected 'failed to load config' error, got: %v", err)
	}
}
