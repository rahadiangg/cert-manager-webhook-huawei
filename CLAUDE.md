## Project Overview

This is a cert-manager webhook that integrates cert-manager with Huawei Cloud DNS to solve ACME DNS-01 challenges. When cert-manager requests a certificate, this webhook creates/deletes TXT records in Huawei Cloud DNS for validation.

## Development Commands

```bash
# Build Docker image
make build

# Run tests (requires setup-envtest)
make test

# Generate Kubernetes manifests from Helm chart
make rendered-manifest.yaml

# Clean build artifacts
make clean
```

## Architecture

### Entry Point
- `main.go` - Sets `GROUP_NAME` env var and registers `HuaweiCloudSolver` with cert-manager's webhook server

### Core Components (`pkg/huaweicloud/`)

1. **solver.go** - Implements cert-manager's `webhook.Solver` interface:
   - `Present()` - Creates TXT record for ACME challenge
   - `CleanUp()` - Deletes TXT record after validation
   - `Initialize()` - Sets up Kubernetes client
   - `getCredentials()` - Retrieves AK/SK from Kubernetes Secrets

2. **dns.go** - Huawei Cloud DNS SDK wrapper:
   - `NewDNSClient()` - Creates authenticated DNS client
   - `CreateTXTRecord()` - Creates TXT record with proper quoting
   - `DeleteTXTRecord()` - Deletes record by matching value (idempotent)
   - `getZoneID()` - Resolves zone name to zone ID
   - `extractRecordName()` - Extracts record name from FQDN

3. **config.go** - Configuration structures:
   - `HuaweiCloudConfig` - JSON config from ClusterIssuer (region, projectId, zoneName, credential refs)
   - `SecretKeySelector` - Reference to Kubernetes Secret key
   - `loadConfig()` - Validates required fields

### Deployment

- Helm chart in `deploy/huawei-webhook/`
- `groupName` (set in values.yaml) must match ClusterIssuer config
- Webhook runs in `cert-manager` namespace
- Requires RBAC for reading Secrets

### Configuration Flow

1. ClusterIssuer references webhook with `groupName` and `solverName: huawei-solver`
2. `config` section contains Huawei Cloud settings (region, projectId, zoneName)
3. `akSecretRef`/`skSecretRef` point to Kubernetes Secrets with credentials
4. On Present/CleanUp, solver loads config, fetches credentials, creates/deletes TXT records

### Important Notes

- TXT record values must be quoted for Huawei Cloud API (handled in `CreateTXTRecord`)
- Zone name matching handles both trailing-dot and no-trailing-dot formats
- `DeleteTXTRecord` is idempotent - returns nil if record not found
- Region mapping in `getRegionID()` covers common Huawei Cloud regions
