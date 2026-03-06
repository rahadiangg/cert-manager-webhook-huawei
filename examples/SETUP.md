# Complete Setup Example for cert-manager-webhook-huawei

This guide provides a complete end-to-end example for setting up the Huawei Cloud DNS webhook for cert-manager.

## Prerequisites

### 1. Huawei Cloud Setup

**Create IAM User with DNS Permissions:**

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

## Step-by-Step Installation

### Step 1: Build and Push Docker Image

```bash
# Build the image
docker build -t yourregistry.com/cert-manager-webhook-huawei:v1.0.0 .

# Push to your registry
docker push yourregistry.com/cert-manager-webhook-huawei:v1.0.0
```

### Step 2: Install cert-manager (if not already installed)

```bash
# Add jetstack repo
helm repo add jetstack https://charts.jetstack.io
helm repo update

# Install cert-manager
kubectl create namespace cert-manager
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version v1.19.0 \
  --set installCRDs=true
```

### Step 3: Install the Huawei Webhook

```bash
# Install via Helm
helm install huawei-webhook ./deploy/huawei-webhook \
  --namespace cert-manager \
  --set image.repository=yourregistry.com/cert-manager-webhook-huawei \
  --set image.tag=v1.0.0 \
  --set groupName=acme.yourdomain.com
```

**Important:** Replace `acme.yourdomain.com` with a domain you own.

### Step 4: Create Credentials Secret

Edit `examples/01-huawei-credentials-secret.yaml` with your actual credentials:

```bash
kubectl apply -f examples/01-huawei-credentials-secret.yaml
```

Or apply directly:
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

### Step 5: Create ClusterIssuer

**For testing (recommended first):**
```bash
# Edit examples/03-staging-clusterissuer.yaml with your details
kubectl apply -f examples/03-staging-clusterissuer.yaml
```

**For production:**
```bash
# Edit examples/02-clusterissuer.yaml with your details
kubectl apply -f examples/02-clusterissuer.yaml
```

### Step 6: Create Certificate

```bash
# For wildcard certificate
kubectl apply -f examples/04-certificate-wildcard.yaml

# Or for single domain
kubectl apply -f examples/05-certificate-single.yaml
```

### Step 7: Verify Certificate Status

```bash
# Check certificate status
kubectl describe certificate example-com-wildcard -n default

# Watch certificate events
kubectl get certificate example-com-wildcard -n default -w
```

Expected output for successful certificate:
```
Status:
  Conditions:
    Type:          Ready
    Status:        True
    Message:       Certificate is up to date and has not expired
```

## Configuration Example Files

| File | Purpose |
|------|---------|
| `01-huawei-credentials-secret.yaml` | Huawei Cloud AK/SK credentials |
| `02-clusterissuer.yaml` | Production Let's Encrypt issuer |
| `03-staging-clusterissuer.yaml` | Staging Let's Encrypt issuer (for testing) |
| `04-certificate-wildcard.yaml` | Wildcard certificate example |
| `05-certificate-single.yaml` | Single domain certificate example |
| `06-ingress-example.yaml` | Ingress using the certificate |

## Common Issues and Solutions

### Issue: "failed to list zones"

**Cause:** Missing or incorrect IAM permissions

**Solution:**
- Verify IAM policy includes `dns:zone:list`
- Check policy is attached to your IAM user
- Verify Project ID is correct

### Issue: "zone not found"

**Cause:** Incorrect zoneName or projectId

**Solution:**
```bash
# Check your zone name
kubectl get clusterissuer letsencrypt-huawei-staging -o yaml
```

### Issue: "Unauthorized"

**Cause:** Invalid AK/SK

**Solution:**
```bash
# Verify credentials in secret
kubectl get secret huawei-cloud-credentials -n cert-manager -o yaml
```

### Issue: "dry run failed"

**Cause:** Webhook not responding

**Solution:**
```bash
# Check webhook logs
kubectl logs -n cert-manager -l app=huawei-webhook

# Check webhook is running
kubectl get pods -n cert-manager -l app=huawei-webhook

# Check API service
kubectl get apiservice | grep huawei
```

## Testing Checklist

- [ ] IAM user created with DNS permissions
- [ ] AK/SK saved securely
- [ ] Project ID and Region noted
- [ ] Docker image built and pushed
- [ ] cert-manager installed
- [ ] Webhook installed with correct groupName
- [ ] Credentials secret created
- [ ] ClusterIssuer created (start with staging!)
- [ ] Test certificate created
- [ ] Certificate issued successfully
- [ ] Switch to production ClusterIssuer

## Production Tips

1. **Always test with staging first** - Let's Encrypt has strict rate limits
2. **Monitor certificate expiration** - Set up alerts for certificate renewals
3. **Keep credentials secure** - Use Kubernetes Secrets, never commit to git
4. **Regular updates** - Keep cert-manager and webhook updated
5. **Backup your certificates** - Export secrets regularly

## Useful Commands

```bash
# List all certificates
kubectl get certificate -A

# Describe certificate for details
kubectl describe certificate <name> -n <namespace>

# Get certificate secret
kubectl get secret <secret-name> -n <namespace> -o yaml

# Check webhook logs
kubectl logs -n cert-manager -l app=huawei-webhook -f

# Restart webhook
kubectl rollout restart deployment huawei-webhook -n cert-manager

# Delete certificate to force renewal
kubectl delete certificate <name> -n <namespace>
```
