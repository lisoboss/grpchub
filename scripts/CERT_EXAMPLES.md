# GrpcHub Certificate Generation Examples

## Quick Reference

### Default Configuration (Development)
```bash
# Generates certificates for localhost, 127.0.0.1, ::1
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash
```

### Production with Custom Domain
```bash
# Single domain
GRPCHUB_DOMAIN=api.example.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Using command line arguments
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash -s -- --domain api.example.com
```

### Production with Custom IP
```bash
# Static IP address
GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Using command line arguments
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash -s -- --ip 192.168.1.100
```

## Complete Installation Examples

### Local Development
```bash
# Install GrpcHub with default certificates
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/install.sh | sh
```

### Production with Domain
```bash
# Install GrpcHub with custom domain certificate
GRPCHUB_DOMAIN=grpc.mycompany.com curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/install.sh | sh
```

### Production with Static IP
```bash
# Install GrpcHub with custom IP certificate
GRPCHUB_IP=203.0.113.10 curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/install.sh | sh
```

## Security Comparison

| Configuration | Subject Alternative Names (SAN) | Security Level | Use Case |
|---------------|--------------------------------|----------------|----------|
| **Default** | `localhost`, `*.localhost`, `127.0.0.1`, `::1`, `0.0.0.0` | Development | Local testing, development environments |
| **Custom Domain** | `example.com`, `*.example.com` | Production | Public servers with domain names |
| **Custom IP** | `192.168.1.100` | Production | Servers with static IP addresses |

## Certificate Files Generated

After running any generation method, you'll have:

```
certs/
├── ca.crt          # Certificate Authority (for client verification)
├── ca.key          # CA Private Key (keep secure!)
├── server.crt      # Server Certificate
├── server.key      # Server Private Key
├── client.crt      # Client Certificate
├── client.key      # Client Private Key
├── server.pem      # Combined server cert + key + CA (use with --pem)
└── client.pem      # Combined client cert + key + CA
```

## Verification Commands

### Check Certificate Details
```bash
# View server certificate details
openssl x509 -in certs/server.crt -text -noout

# Check Subject Alternative Names
openssl x509 -in certs/server.crt -text -noout | grep -A 1 "Subject Alternative Name"

# Verify certificate chain
openssl verify -CAfile certs/ca.crt certs/server.crt
```

### Test Certificate Validity
```bash
# Test with specific hostname
openssl s_client -connect localhost:50055 -servername localhost -CAfile certs/ca.crt

# Test with custom domain (replace with your domain)
openssl s_client -connect api.example.com:50055 -servername api.example.com -CAfile certs/ca.crt
```

## Docker Compose Usage

### With Default Certificates
```bash
curl -O https://raw.githubusercontent.com/lisoboss/grpchub/main/deploy/docker-compose.ghcr.yaml
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash
docker-compose -f docker-compose.ghcr.yaml up -d
```

### With Custom Domain
```bash
curl -O https://raw.githubusercontent.com/lisoboss/grpchub/main/deploy/docker-compose.ghcr.yaml
GRPCHUB_DOMAIN=grpc.mycompany.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
docker-compose -f docker-compose.ghcr.yaml up -d
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `GRPCHUB_DOMAIN` | Custom domain name | `api.example.com` |
| `GRPCHUB_IP` | Custom IP address | `192.168.1.100` |

## Common Use Cases

### Development Team
```bash
# Each developer runs locally
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash
```

### Staging Environment
```bash
# Staging server with internal domain
GRPCHUB_DOMAIN=grpchub-staging.internal bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
```

### Production Environment
```bash
# Production server with public domain
GRPCHUB_DOMAIN=grpc-api.company.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
```

### Kubernetes/Container Deployment
```bash
# Generate certificates for service name
GRPCHUB_DOMAIN=grpchub-service.default.svc.cluster.local bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
```

## Security Best Practices

1. **Use Custom Certificates in Production**
   - Never use default certificates in production
   - Specify exact domain or IP for your server

2. **Protect Private Keys**
   - Keep `ca.key` and `server.key` secure
   - Set appropriate file permissions (600 or 640)

3. **Certificate Rotation**
   - Certificates expire after 365 days
   - Set up renewal process for production

4. **Network Security**
   - Custom certificates are restricted to specified domains/IPs only
   - This prevents certificate misuse on other domains

## Troubleshooting

### Certificate Validation Errors
```bash
# Check if certificate matches your domain/IP
openssl x509 -in certs/server.crt -text -noout | grep -A 1 "Subject Alternative Name"

# Regenerate with correct domain/IP
GRPCHUB_DOMAIN=correct-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
```

### Permission Issues
```bash
# Fix certificate permissions
chmod 600 certs/*.key
chmod 644 certs/*.crt certs/*.pem
```

### Docker Volume Issues
```bash
# Ensure certificates are in correct location
ls -la certs/server.pem
# Should show the server.pem file with proper permissions
```