package huaweicloud

import (
	"encoding/json"
	"fmt"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// SecretKeySelector selects a key from a Secret.
type SecretKeySelector struct {
	// Name is the name of the Secret
	Name string `json:"name"`

	// Key is the key in the Secret data
	Key string `json:"key"`
}

// HuaweiCloudConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
type HuaweiCloudConfig struct {
	// Region is the Huawei Cloud region (e.g., cn-north-4, cn-southwest-2)
	Region string `json:"region"`

	// ProjectID is the Huawei Cloud project ID
	ProjectID string `json:"projectId"`

	// ZoneName is the DNS zone name (e.g., example.com)
	ZoneName string `json:"zoneName"`

	// AKSecretRef is a reference to a Secret containing the Access Key ID
	AKSecretRef SecretKeySelector `json:"akSecretRef"`

	// SKSecretRef is a reference to a Secret containing the Secret Access Key
	SKSecretRef SecretKeySelector `json:"skSecretRef"`
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (HuaweiCloudConfig, error) {
	cfg := HuaweiCloudConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, fmt.Errorf("configuration must be provided")
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	// Validate required fields
	if cfg.Region == "" {
		return cfg, fmt.Errorf("region is required")
	}
	if cfg.ProjectID == "" {
		return cfg, fmt.Errorf("projectId is required")
	}
	if cfg.ZoneName == "" {
		return cfg, fmt.Errorf("zoneName is required")
	}
	if cfg.AKSecretRef.Name == "" {
		return cfg, fmt.Errorf("akSecretRef.name is required")
	}
	if cfg.AKSecretRef.Key == "" {
		return cfg, fmt.Errorf("akSecretRef.key is required")
	}
	if cfg.SKSecretRef.Name == "" {
		return cfg, fmt.Errorf("skSecretRef.name is required")
	}
	if cfg.SKSecretRef.Key == "" {
		return cfg, fmt.Errorf("skSecretRef.key is required")
	}

	return cfg, nil
}
