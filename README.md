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

1. Go to **IAM Console** → **Users** → **Create User**
   - Username: `cert-manager-webhook`
   - Access Mode: **Programmatic Access**
   - Save the **Access Key ID (AK)** and **Secret Access Key (SK)**

2. Create a custom policy:
```json
{
    "Version": "1.1",
    "Statement": [{
        "Effect": "Allow",
        "Action": [
            "dns:zone:list",
            "dns:recordset:list",
            "dns:recordset:create",
            "dns:recordset:delete"
        ]
    }]
}
```

3. Attach the policy to your IAM user

### 2. Get Your Huawei Cloud Details

| Item | How to Find |
|------|-------------|
| **Project ID** | DNS Console → click your domain → check URL, or EPS → Enterprise Projects |
| **Region** | Top-right corner of Huawei Cloud console (e.g., `cn-north-4`) |
| **Zone Name** | Your domain name (e.g., `example.com`) |

### 3. Build and Push Docker Image

```bash
docker build -t yourregistry.com/cert-manager-webhook-huawei:v1.0.0 .
docker push yourregistry.com/cert-manager-webhook-huawei:v1.0.0
```

### 4. Install the Webhook

```bash
helm install huawei-webhook ./deploy/huawei-webhook \
  --namespace cert-manager \
  --set image.repository=yourregistry.com/cert-manager-webhook-huawei \
  --set image.tag=v1.0.0 \
  --set groupName=acme.yourdomain.com
```

**Important:** Replace `acme.yourdomain.com` with a domain you own.

### 5. Create Credentials Secret

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: huawei-cloud-credentials
  namespace: cert-manager
type: Opaque
stringData:
  access-key-id: "YOUR_ACCESS_KEY_ID"
  secret-access-key: "YOUR_SECRET_ACCESS_KEY"
EOF
```

### 6. Create ClusterIssuer

Start with staging (recommended for testing):

```bash
kubectl apply -f examples/03-staging-clusterissuer.yaml
```

For production:
```bash
kubectl apply -f examples/02-clusterissuer.yaml
```

Edit the file with your details:
- `region`: Your Huawei Cloud region
- `projectId`: Your project ID
- `zoneName`: Your domain name
- `groupName`: Must match what you set in Helm install
- `akSecretRef`/`skSecretRef`: References to your credentials secret

### 7. Create Certificate

```bash
# For wildcard certificate
kubectl apply -f examples/04-certificate-wildcard.yaml

# Or for single domain
kubectl apply -f examples/05-certificate-single.yaml
```

### 8. Verify

```bash
kubectl describe certificate example-com-wildcard -n default
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
| `02-clusterissuer.yaml` | Production Let's Encrypt issuer |
| `03-staging-clusterissuer.yaml` | Staging Let's Encrypt issuer (for testing) |
| `04-certificate-wildcard.yaml` | Wildcard certificate example |
| `05-certificate-single.yaml` | Single domain certificate example |
| `06-ingress-example.yaml` | Ingress using the certificate |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `failed to list zones` | Check IAM permissions include `dns:zone:list` |
| `zone not found` | Verify `zoneName` and `projectId` in ClusterIssuer |
| `Unauthorized` | Verify AK/SK in the secret are correct |
| `dry run failed` | Check webhook is running: `kubectl logs -n cert-manager -l app=huawei-webhook` |

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

## How it works

1. cert-manager creates an ACME challenge for your domain
2. cert-manager calls this webhook
3. The webhook creates a TXT record in Huawei Cloud DNS
4. Let's Encrypt validates the TXT record
5. The webhook deletes the TXT record
6. Let's Encrypt issues the certificate

## License

MIT
