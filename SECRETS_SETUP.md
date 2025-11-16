# Secrets Setup

## API Key Configuration

The network visualizer uses AI-powered insights which require an OpenRouter API key. This key is stored as a Kubernetes secret.

### Create the Secret

```bash
# Replace YOUR_API_KEY_HERE with your actual OpenRouter API key
kubectl create secret generic openrouter-api-key \
  --from-literal=api-key=YOUR_API_KEY_HERE \
  -n network-visualizer
```

### Verify the Secret

```bash
kubectl get secret openrouter-api-key -n network-visualizer
```

### Update the Secret

```bash
# Delete old secret
kubectl delete secret openrouter-api-key -n network-visualizer

# Create new secret with updated key
kubectl create secret generic openrouter-api-key \
  --from-literal=api-key=YOUR_NEW_API_KEY \
  -n network-visualizer

# Restart deployment to pick up new secret
kubectl rollout restart deployment/network-visualizer -n network-visualizer
```

### Running Without AI Features

The AI API key is optional. If not provided, the visualizer will work but AI-powered insights will be disabled. The secret is marked as `optional: true` in the deployment configuration.

## Security Best Practices

1. **Never commit API keys to git**
2. **Use Kubernetes secrets** for sensitive data
3. **Rotate keys regularly**
4. **Limit secret access** using RBAC
5. **Consider using a secrets manager** (e.g., HashiCorp Vault, AWS Secrets Manager) for production

## Alternative: Using External Secrets Operator

For production environments, consider using the External Secrets Operator:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
  namespace: network-visualizer
spec:
  provider:
    vault:
      server: "https://vault.example.com"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "network-visualizer"
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: openrouter-api-key
  namespace: network-visualizer
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: openrouter-api-key
  data:
  - secretKey: api-key
    remoteRef:
      key: network-visualizer/openrouter
      property: api-key
```
