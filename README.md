# cert-manager Webhook for Huawei Cloud DNS

A cert-manager webhook for solving DNS-01 challenges using Huawei Cloud DNS. Get automated SSL certificates from Let's Encrypt for domains hosted on Huawei Cloud DNS.

## What is this?

This webhook integrates [cert-manager](https://cert-manager.io) with Huawei Cloud DNS to automatically obtain SSL certificates using DNS validation. When you request a certificate, the webhook creates a TXT record in your Huawei Cloud DNS, Let's Encrypt validates it, and then the record is cleaned up automatically.

## What you need

- Kubernetes cluster with cert-manager installed
- Huawei Cloud account with DNS service enabled
- Huawei Cloud Access Key (AK) and Secret Key (SK)

## Quick Start

### 1. Create IAM User with DNS Permissions

Go to **IAM Console** → **Users** → **Create User**:
- Username: `cert-manager-webhook`
- Access Mode: **Programmatic Access**
- Save the **Access Key ID (AK)** and **Secret Access Key (SK)**

Create a custom policy:
```json
{
    "Version": "1.1",
    "Statement": [{
        "Effect": "Allow",
        "Action": [
            "dns:zone:list",
            "dns:recordset:list",
            "dns:recordset:create",
            "dns:recordset:update",
            "dns:recordset:delete"
        ]
    }]
}
```

Attach the policy to your IAM user.

### 2. Install the Webhook

```bash
helm install huawei-webhook ./deploy/huawei-webhook \
  --namespace cert-manager \
  --set image.repository=yourregistry.com/cert-manager-webhook-huawei \
  --set image.tag=1.0.0 \
  --set groupName=acme.yourdomain.com
```

**Important:** Replace `acme.yourdomain.com` with a domain you own.

### Optional: Configure Logging

You can customize logging verbosity and format:

```bash
helm install huawei-webhook ./deploy/huawei-webhook \
  --namespace cert-manager \
  --set image.repository=yourregistry.com/cert-manager-webhook-huawei \
  --set image.tag=1.0.0 \
  --set groupName=acme.yourdomain.com \
  --set logLevel=debug \
  --set logFormat=json
```

**Log Levels:**
- `debug` - Detailed logging for troubleshooting
- `info` (default) - Standard informational messages
- `warn` - Warning messages only
- `error` - Error messages only

**Log Formats:**
- `text` (default) - Human-readable text format
- `json` - JSON format for log aggregation systems

### 4. Apply Example Files

```bash
# 1. Create credentials secret (edit with your Huawei Cloud AK/SK first)
kubectl apply -f examples/01-huawei-credentials-secret.yaml

# 2. Create staging ClusterIssuer (edit with your region, projectId, zoneName)
kubectl apply -f examples/02-staging-clusterissuer.yaml

# 3. Create certificate
kubectl apply -f examples/03-certificate-wildcard.yaml
```

### 5. Verify

```bash
kubectl describe certificate example-wildcard-certificate -n cert-manager
```

Expected output:
```
Status:
  Conditions:
    Type:          Ready
    Status:        True
    Message:       Certificate is up to date and has not expired
```

## Example Files

| File | Purpose |
|------|---------|
| `01-huawei-credentials-secret.yaml` | Huawei Cloud AK/SK credentials |
| `02-staging-clusterissuer.yaml` | Let's Encrypt staging issuer (for testing) |
| `03-certificate-wildcard.yaml` | Wildcard certificate example |

## Configuration

Edit the example files with your details:

### ClusterIssuer (`02-staging-clusterissuer.yaml`)

- `region`: Use `cn-north-4` (Huawei Cloud DNS is a global service, and this region provides the most stable API endpoint)
- `projectId`: Your project ID (found in DNS Console URL or EPS → Enterprise Projects)
- `zoneName`: Your domain name
- `groupName`: Must match what you set in Helm install
- `email`: Your email for Let's Encrypt notifications

**Note:** While Huawei Cloud DNS supports multiple regions, `cn-north-4` is recommended as it's the primary region for the DNS service and provides the most reliable API access.

### Certificate (`03-certificate-wildcard.yaml`)

- `dnsNames`: The domains you want certificates for

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `failed to list zones` | Check IAM permissions include `dns:zone:list` |
| `zone not found` | Verify `zoneName` and `projectId` in ClusterIssuer |
| `Unauthorized` | Verify AK/SK in the secret are correct |
| `DNS.1802 No permissions` | Add `dns:recordset:update` to IAM policy (required for updating existing TXT records) |
| `dial tcp: lookup region` | Ensure region is set to `cn-north-4` for DNS service |
| `no such host` | Verify you're using `cn-north-4` region for DNS API |
| `dry run failed` | Check webhook is running: `kubectl logs -n cert-manager -l app=huawei-webhook` |
| `record already exists` | The webhook will automatically update existing TXT records |

### Debug Logging

For detailed logging to troubleshoot issues, reinstall the webhook with debug level:

```bash
helm upgrade huawei-webhook ./deploy/huawei-webhook \
  --namespace cert-manager \
  --reuse-values \
  --set logLevel=debug
```

Then view the logs:
```bash
kubectl logs -n cert-manager -l app=huawei-webhook -f
```

### Log Levels Explained

- **debug** - Shows all operations including DNS API calls, record lookups, and internal processing
- **info** (default) - Shows successful operations (records created/deleted) and errors
- **warn** - Shows only warnings and errors
- **error** - Shows only errors

## Useful Commands

```bash
# List all certificates
kubectl get certificate -A

# Check webhook logs
kubectl logs -n cert-manager -l app=huawei-webhook -f

# Verify API registration
kubectl get apiservice | grep huawei

# Restart webhook
kubectl rollout restart deployment huawei-webhook -n cert-manager
```

## License

MIT
