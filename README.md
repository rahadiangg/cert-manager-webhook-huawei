# cert-manager Webhook for Huawei Cloud DNS

A cert-manager webhook for solving DNS-01 challenges using Huawei Cloud DNS.

## Overview

This webhook integrates cert-manager with Huawei Cloud DNS to enable automated ACME DNS-01 challenge validation. It allows you to obtain wildcard SSL certificates from Let's Encrypt (or other ACME CAs) for domains hosted on Huawei Cloud DNS.

## Features

- Automatic DNS TXT record creation for ACME challenges
- Automatic cleanup of challenge records
- Support for concurrent challenges for the same domain
- Credentials stored securely in Kubernetes Secrets
- Compatible with cert-manager v1.19+

## Prerequisites

- Kubernetes cluster with cert-manager installed
- Huawei Cloud account with DNS service enabled
- Huawei Cloud Access Key (AK) and Secret Key (SK)
- IAM user with minimum DNS permissions (see below)

## IAM Permissions

The webhook requires an IAM user with minimum DNS permissions to manage TXT records for ACME challenges.

### Required Permissions

Create a custom IAM policy with the following permissions:

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "dns:zone:list",
                "dns:recordset:list",
                "dns:recordset:create",
                "dns:recordset:delete"
            ]
        }
    ]
}
```

### Creating IAM User

1. Go to **Identity and Access Management (IAM)** → **Users** → **Create User**
2. Set username (e.g., `cert-manager-webhook`)
3. Select **Access Mode - Programmatic Access**
4. Save the **Access Key ID (AK)** and **Secret Access Key (SK)** securely
5. Attach the custom policy created above

### Security Best Practice

- Use a dedicated IAM user for the webhook only
- Grant minimum required permissions (no admin access)
- Rotate credentials periodically
- Store credentials in Kubernetes Secrets only

### Finding Your Project ID

1. Go to **DNS Console** → click your domain name
2. The project ID is shown in the URL or domain details
3. Alternatively: **EPS** → **Enterprise Projects** → find your project ID

### Finding Your Region

The region is shown in the top-right corner of the Huawei Cloud console (e.g., `cn-north-4` for Beijing).

## Quick Start

1. **Create IAM user** with permissions above → save AK/SK
2. **Create credentials secret** in Kubernetes
3. **Install webhook** via Helm (set `groupName` to your domain)
4. **Create ClusterIssuer** with your Huawei Cloud DNS settings
5. **Create Certificate** resource for your domain

## Installation

### 1. Build and push the Docker image

```bash
docker build -t yourregistry/cert-manager-webhook-huawei:v1.0.0 .
docker push yourregistry/cert-manager-webhook-huawei:v1.0.0
```

### 2. Install the webhook with Helm

```bash
helm install huawei-webhook ./deploy/huawei-webhook \
  --set image.repository=yourregistry/cert-manager-webhook-huawei \
  --set image.tag=v1.0.0 \
  --set groupName=acme.yourdomain.com \
  --namespace cert-manager
```

**Important:** Set `groupName` to a domain you own. This is used to register the webhook API.

### 3. Create Huawei Cloud credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: huawei-cloud-credentials
  namespace: cert-manager
type: Opaque
stringData:
  access-key-id: <your-huawei-ak>
  secret-access-key: <your-huawei-sk>
```

Apply the secret:
```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: huawei-cloud-credentials
  namespace: cert-manager
type: Opaque
stringData:
  access-key-id: YOUR_ACCESS_KEY_ID
  secret-access-key: YOUR_SECRET_ACCESS_KEY
EOF
```

### 4. Create a ClusterIssuer

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-huawei
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-huawei
    solvers:
    - dns01:
        webhook:
          groupName: acme.yourdomain.com
          solverName: huawei-solver
          config:
            region: cn-north-4
            projectId: "your-project-id"
            zoneName: example.com
            akSecretRef:
              name: huawei-cloud-credentials
              key: access-key-id
            skSecretRef:
              name: huawei-cloud-credentials
              key: secret-access-key
```

## Configuration

### ClusterIssuer Configuration

| Parameter | Description | Example |
|-----------|-------------|---------|
| `region` | Huawei Cloud region | `cn-north-4`, `cn-southwest-2` |
| `projectId` | Huawei Cloud project ID | `"your-project-id"` |
| `zoneName` | DNS zone name | `example.com` |
| `akSecretRef.name` | Secret containing AK | `huawei-cloud-credentials` |
| `akSecretRef.key` | Key in Secret for AK | `access-key-id` |
| `skSecretRef.name` | Secret containing SK | `huawei-cloud-credentials` |
| `skSecretRef.key` | Key in Secret for SK | `secret-access-key` |

### Supported Regions

- `cn-north-4` - Beijing
- `cn-north-1` - Beijing (alt)
- `cn-south-1` - Guangzhou
- `cn-southwest-2` - Chengdu
- `ap-southeast-1` - Hong Kong
- `ap-southeast-2` - Bangkok
- `ap-southeast-3` - Singapore

## Usage

Once installed and configured, use the webhook in your Certificate resources:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com-wildcard
  namespace: default
spec:
  secretName: example-com-tls
  issuerRef:
    name: letsencrypt-huawei
    kind: ClusterIssuer
  dnsNames:
  - "*.example.com"
  - example.com
```

## Development

### Running tests

```bash
go test -v ./...
```

### Building locally

```bash
go build -o webhook
./webhook --help
```

## Troubleshooting

### Check webhook logs

```bash
kubectl logs -n cert-manager -l app=huawei-webhook
```

### Verify API registration

```bash
kubectl get apiservice | grep huawei
```

### Check certificate status

```bash
kubectl get certificate -A
kubectl describe certificate <certificate-name>
```

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to list zones` | Missing `dns:zone:list` permission | Add to IAM policy |
| `failed to create TXT record` | Missing `dns:recordset:create` permission | Add to IAM policy |
| `zone not found` | Incorrect `zoneName` or `projectId` | Check ClusterIssuer config |
| `failed to create credentials` | Invalid AK/SK | Verify credentials in Secret |
| `Unauthorized` | IAM policy not attached | Attach policy to IAM user |

## How it works

1. cert-manager creates an ACME challenge for your domain
2. cert-manager calls this webhook via the configured groupName
3. The webhook creates a TXT record in Huawei Cloud DNS
4. Let's Encrypt validates the TXT record
5. The webhook deletes the TXT record
6. Let's Encrypt issues the certificate

## Security

- Credentials are stored in Kubernetes Secrets
- RBAC restricts Secret access to the webhook only
- Webhook communication uses TLS
- No credentials are logged

## License

MIT License

## References

- [Huawei Cloud DNS API](https://support.huaweicloud.com/api-dns/dns_api_64001.html)
- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [cert-manager Webhook Documentation](https://cert-manager.io/docs/contributing/webhook/)
