package huaweicloud

import (
	"encoding/json"
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfgJSON     *extapi.JSON
		wantErr     bool
		errContains string
		wantConfig  HuaweiCloudConfig
	}{
		{
			name:    "nil config",
			cfgJSON: nil,
			wantErr: true,
			errContains: "configuration must be provided",
		},
		{
			name: "invalid JSON",
			cfgJSON: &extapi.JSON{
				Raw: []byte("{invalid json"),
			},
			wantErr:     true,
			errContains: "error decoding solver config",
		},
		{
			name: "empty config JSON",
			cfgJSON: &extapi.JSON{
				Raw: []byte("{}"),
			},
			wantErr:     true,
			errContains: "region is required",
		},
		{
			name: "missing region",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"projectId": "test-project",
					"zoneName": "example.com",
					"akSecretRef": {"name": "ak-secret", "key": "access-key"},
					"skSecretRef": {"name": "sk-secret", "key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "region is required",
		},
		{
			name: "missing projectId",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"zoneName": "example.com",
					"akSecretRef": {"name": "ak-secret", "key": "access-key"},
					"skSecretRef": {"name": "sk-secret", "key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "projectId is required",
		},
		{
			name: "missing zoneName",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project",
					"akSecretRef": {"name": "ak-secret", "key": "access-key"},
					"skSecretRef": {"name": "sk-secret", "key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "zoneName is required",
		},
		{
			name: "missing akSecretRef name",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project",
					"zoneName": "example.com",
					"akSecretRef": {"key": "access-key"},
					"skSecretRef": {"name": "sk-secret", "key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "akSecretRef.name is required",
		},
		{
			name: "missing akSecretRef key",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project",
					"zoneName": "example.com",
					"akSecretRef": {"name": "ak-secret"},
					"skSecretRef": {"name": "sk-secret", "key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "akSecretRef.key is required",
		},
		{
			name: "missing skSecretRef name",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project",
					"zoneName": "example.com",
					"akSecretRef": {"name": "ak-secret", "key": "access-key"},
					"skSecretRef": {"key": "secret-key"}
				}`),
			},
			wantErr:     true,
			errContains: "skSecretRef.name is required",
		},
		{
			name: "missing skSecretRef key",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project",
					"zoneName": "example.com",
					"akSecretRef": {"name": "ak-secret", "key": "access-key"},
					"skSecretRef": {"name": "sk-secret"}
				}`),
			},
			wantErr:     true,
			errContains: "skSecretRef.key is required",
		},
		{
			name: "valid config",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "cn-north-4",
					"projectId": "test-project-id",
					"zoneName": "example.com",
					"akSecretRef": {"name": "huawei-ak-secret", "key": "access-key-id"},
					"skSecretRef": {"name": "huawei-sk-secret", "key": "secret-access-key"}
				}`),
			},
			wantErr: false,
			wantConfig: HuaweiCloudConfig{
				Region:    "cn-north-4",
				ProjectID: "test-project-id",
				ZoneName:  "example.com",
				AKSecretRef: SecretKeySelector{
					Name: "huawei-ak-secret",
					Key:  "access-key-id",
				},
				SKSecretRef: SecretKeySelector{
					Name: "huawei-sk-secret",
					Key:  "secret-access-key",
				},
			},
		},
		{
			name: "valid config with different region",
			cfgJSON: &extapi.JSON{
				Raw: []byte(`{
					"region": "ap-southeast-1",
					"projectId": "project-123",
					"zoneName": "test.org",
					"akSecretRef": {"name": "secret1", "key": "key1"},
					"skSecretRef": {"name": "secret2", "key": "key2"}
				}`),
			},
			wantErr: false,
			wantConfig: HuaweiCloudConfig{
				Region:    "ap-southeast-1",
				ProjectID: "project-123",
				ZoneName:  "test.org",
				AKSecretRef: SecretKeySelector{
					Name: "secret1",
					Key:  "key1",
				},
				SKSecretRef: SecretKeySelector{
					Name: "secret2",
					Key:  "key2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadConfig(tt.cfgJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errContains != "" {
					errStr := err.Error()
					if !contains(errStr, tt.errContains) {
						t.Errorf("loadConfig() error = %v, expected to contain %v", err, tt.errContains)
					}
				}
				return
			}
			if !tt.wantErr && !configEqual(got, tt.wantConfig) {
				t.Errorf("loadConfig() = %v, want %v", got, tt.wantConfig)
			}
		})
	}
}

func TestSecretKeySelectorJSON(t *testing.T) {
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
			name: "valid selector with empty values",
			json: `{"name": "", "key": "my-key"}`,
			want: SecretKeySelector{
				Name: "",
				Key:  "my-key",
			},
			wantErr: false,
		},
		{
			name: "valid selector with empty key",
			json: `{"name": "my-secret", "key": ""}`,
			want: SecretKeySelector{
				Name: "my-secret",
				Key:  "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SecretKeySelector
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !secretKeySelectorEqual(got, tt.want) {
				t.Errorf("JSON unmarshal got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHuaweiCloudConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      HuaweiCloudConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "all required fields present",
			config: HuaweiCloudConfig{
				Region:    "cn-north-4",
				ProjectID: "project-123",
				ZoneName:  "example.com",
				AKSecretRef: SecretKeySelector{
					Name: "ak-secret",
					Key:  "ak-key",
				},
				SKSecretRef: SecretKeySelector{
					Name: "sk-secret",
					Key:  "sk-key",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JSON and back to test validation
			jsonData, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("failed to marshal config: %v", err)
			}

			_, err = loadConfig(&extapi.JSON{Raw: jsonData})
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func configEqual(a, b HuaweiCloudConfig) bool {
	return a.Region == b.Region &&
		a.ProjectID == b.ProjectID &&
		a.ZoneName == b.ZoneName &&
		a.AKSecretRef.Name == b.AKSecretRef.Name &&
		a.AKSecretRef.Key == b.AKSecretRef.Key &&
		a.SKSecretRef.Name == b.SKSecretRef.Name &&
		a.SKSecretRef.Key == b.SKSecretRef.Key
}

func secretKeySelectorEqual(a, b SecretKeySelector) bool {
	return a.Name == b.Name && a.Key == b.Key
}
