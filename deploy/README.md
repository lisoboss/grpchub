# GrpcHub Deployment Guide

This directory contains all the necessary files for deploying GrpcHub in various environments.

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Make (optional, for convenience commands)

### 1. Direct Download (No Git Clone Required)

```bash
# Download deployment configuration
curl -O https://raw.githubusercontent.com/lisoboss/grpchub/main/deploy/docker-compose.ghcr.yaml

# Generate TLS certificates
# Default (localhost, 127.0.0.1, ::1):
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash

# Production with custom domain (recommended):
# GRPCHUB_DOMAIN=your-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Custom IP address:
# GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Start the server
docker-compose -f docker-compose.ghcr.yaml up -d
```

### 2. Repository-based Deployment

```bash
# Clone repository first
git clone https://github.com/lisoboss/grpchub.git
cd grpchub/deploy

# Generate certificates
make certs

# Start server (build from source or use GHCR)
make run          # Build from source
# or
make ghcr         # Use GHCR image
```

### 3. Verify Deployment

```bash
# For direct download
docker-compose -f docker-compose.ghcr.yaml ps

# For repository-based
make health       # Source build
# or
make health-ghcr  # GHCR image
```

## Prerequisites

### System Requirements

- **Docker & Docker Compose**: Latest stable versions
- **Go**: Version 1.24.2 or later (for client development)
- **Rust**: Latest stable (for building from source)
- **Make**: Optional, for convenience commands
- **OpenSSL**: For certificate generation

## Files Overview

| File | Description |
|------|-------------|
| `docker-compose.yaml` | Production deployment configuration (build from source) |
| `docker-compose.ghcr.yaml` | Production deployment using GitHub Container Registry |
| `docker-compose.dev.yaml` | Development environment setup |
| `Dockerfile` | Multi-stage Docker build |
| `Makefile` | Convenience commands |
| `.env.example` | Environment variables template |
| `gen-certs.sh` | TLS certificate generation script |
| `cert-conf/` | OpenSSL configuration files |

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

Key variables:
- `GRPCHUB_ADDR`: Server listen address (default: `[::1]:50055`)
- `GRPCHUB_PORT`: External port mapping (default: `50055`)
- `RUST_LOG`: Log level (default: `info`)
- `GHCR_IMAGE`: GitHub Container Registry image (default: `ghcr.io/lisoboss/grpchub:latest`)

### TLS Certificates

Certificates must be generated before starting the server. They are created in the `certs/` directory:
- `server.pem` - Server certificate and key (required for server)
- `client.pem` - Client certificate and key (for client applications)
- `ca.crt` - Certificate Authority

**Generation Methods:**
1. **Standalone (no git clone)**:
   - Default: `curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash`
   - Custom domain: `GRPCHUB_DOMAIN=your-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)`
   - Custom IP: `GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)`
2. **Repository-based**: `make certs` (after cloning)
3. **Manual**: See manual certificate generation section below

**Security Considerations:**
- Default certificates include localhost and common development IPs
- Custom domain/IP certificates are restricted to specified values only (more secure)
- For production, always use custom domain/IP configuration


## Deployment Options

### Production Deployment (Build from Source)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Production Deployment (GitHub Container Registry)

```bash
# Pull latest image and start
make quick-start-ghcr

# Or manually:
docker-compose -f docker-compose.ghcr.yaml up -d

# View logs
docker-compose -f docker-compose.ghcr.yaml logs -f

# Stop services
docker-compose -f docker-compose.ghcr.yaml down
```

### Development Environment

```bash
# Start development environment
docker-compose -f docker-compose.dev.yaml up -d

# Access development tools
docker-compose -f docker-compose.dev.yaml exec dev-tools bash

# Hot reload development
docker-compose -f docker-compose.dev.yaml restart grpchub-server-dev
```

## Available Commands

### Make Commands

```bash
make help                   # Show all available commands
make quick-start            # Build and run (requires existing certs)
make quick-start-full       # Generate certs, build, and run (source)
make quick-start-ghcr       # Run GHCR image (requires existing certs)
make quick-start-ghcr-full  # Generate certs and run (GHCR image)
make dev-start              # Development start (requires existing certs)
make dev-start-full         # Generate certs and start development
make certs                  # Generate TLS certificates using Docker
make certs-standalone       # Generate TLS certificates using standalone script
make certs-refresh          # Regenerate certificates
make logs                   # Show production logs
make ghcr-logs              # Show GHCR deployment logs
make health                 # Check service health
make health-ghcr            # Check GHCR deployment health
make clean                  # Clean up containers and volumes
```

## GitHub Container Registry

### Using Pre-built Images

The GHCR deployment option uses pre-built images from GitHub Container Registry:

```bash
# Pull latest image
make ghcr-pull

# Start with GHCR image
make ghcr

# Quick start with latest image
make quick-start-ghcr
```

### Docker Compose Commands

```bash
# Production (build from source)
docker-compose up -d
docker-compose logs -f
docker-compose down

# Production (GHCR image)
docker-compose -f docker-compose.ghcr.yaml up -d
docker-compose -f docker-compose.ghcr.yaml logs -f
docker-compose -f docker-compose.ghcr.yaml down

# Development
docker-compose -f docker-compose.dev.yaml up -d
docker-compose -f docker-compose.dev.yaml logs -f
docker-compose -f docker-compose.dev.yaml down
```

### Image Information

- **Registry**: `ghcr.io/lisoboss/grpchub`
- **Tags**: `latest`, version-specific tags
- **Architecture**: Multi-platform support

### Authentication (if needed)

For private repositories, authenticate with GitHub:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

## Networking

### Default Ports

| Service | Port | Description |
|---------|------|-------------|
| GrpcHub Server | 50055 | Main gRPC service |
| Development Debug | 9090 | Development debugging |

### Network Configuration

- Network Name: `grpchub-network`
- Driver: `bridge`
- Internal communication between services

## Security

### TLS Configuration

- Mutual TLS authentication enabled
- Certificates auto-generated with proper SANs
- Support for custom certificate paths

### Certificate Management

```bash
# View certificate details
openssl x509 -in certs/server.crt -text -noout

# Test certificate validation
openssl verify -CAfile certs/ca.crt certs/server.crt
```

## Monitoring

### Health Checks

Built-in health check endpoint available at the server port.

### Logs

```bash
# View all logs (source build)
make logs

# View GHCR deployment logs
make ghcr-logs

# View specific service logs
docker-compose logs -f grpchub-server

# Development logs
make dev-logs
```

## Troubleshooting

### Common Issues

1. **Certificate Errors**
   ```bash
   # For repository-based deployment
   make certs-refresh

   # For direct download deployment
   curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash
   ```

2. **Port Already in Use**
   ```bash
   # Change port in .env file
   GRPCHUB_PORT=50056
   ```

3. **Permission Issues**
   ```bash
   sudo chown -R $USER:$USER certs/
   ```

4. **Image Pull Issues**
   ```bash
   # Check GitHub Container Registry access
   docker pull ghcr.io/lisoboss/grpchub:latest

   # Use authentication if needed (for private repos)
   echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
   ```

5. **Missing Certificates**
   ```bash
   # Direct download method
   # For direct download deployment (default)
   curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash

   # For production with custom domain
   GRPCHUB_DOMAIN=your-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

   # Repository method
   make certs

   # Check certificate files exist
   ls -la certs/
   ```

### Debug Mode

```bash
# Enable debug logging
export RUST_LOG=debug
docker-compose restart grpchub-server

# Or use development environment
make dev
```

### Container Debugging

```bash
# Access running container (source build)
docker-compose exec grpchub-server sh

# Access GHCR container
docker-compose -f docker-compose.ghcr.yaml exec grpchub-server sh

# View container logs
docker-compose logs --tail=100 grpchub-server

# Check container status
docker-compose ps
```

## Scaling

### Horizontal Scaling

```bash
# Scale server instances (source build)
docker-compose up -d --scale grpchub-server=3

# Scale GHCR instances
docker-compose -f docker-compose.ghcr.yaml up -d --scale grpchub-server=3
```

### Resource Limits

Add to docker-compose.yaml:

```yaml
services:
  grpchub-server:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
```

## Backup and Recovery

### Backup Certificates

```bash
tar -czf grpchub-certs-$(date +%Y%m%d).tar.gz certs/
```

### Recovery

```bash
tar -xzf grpchub-certs-*.tar.gz
docker-compose restart
```

## Production Considerations

1. **Security**
   - Use custom certificates in production
   - Implement proper firewall rules
   - Regular security updates

2. **Performance**
   - Monitor resource usage
   - Tune logging levels
   - Consider connection pooling

3. **Reliability**
   - Implement proper backup strategies
   - Set up monitoring and alerting
   - Plan for disaster recovery

4. **Maintenance**
   - Regular certificate rotation
   - Container image updates
   - Log rotation and cleanup

## Deployment Comparison

| Feature | Direct Download | Repository Source | Repository GHCR | Development |
|---------|----------------|-------------------|-----------------|-------------|
| Setup Time | Fastest | Medium | Fast | Medium |
| Git Required | No | Yes | Yes | Yes |
| Build Time | None | Long | Fast | Medium |
| Customization | None | Full | Limited | Full |
| Updates | Re-download | Manual rebuild | Pull latest | Live reload |
| Certificate Gen | Standalone script | Docker/Make | Docker/Make | Docker/Make |
| Use Case | Production | Custom builds | Quick deployment | Development |

Choose the deployment method that best fits your needs:
- **Direct Download**: Fastest production deployment without Git
- **Repository Source**: When you need custom modifications or latest unreleased features
- **Repository GHCR**: Quick deployment with repository tools and convenience
- **Development**: For active development and debugging

## Certificate Generation Methods

| Method | Command | Requirements | Use Case |
|--------|---------|--------------|----------|
| Standalone (Default) | `curl -sSL https://raw.../gen-certs-standalone.sh \| bash` | curl, openssl | Development, local testing |
| Standalone (Custom Domain) | `GRPCHUB_DOMAIN=example.com bash <(curl -sSL https://raw.../gen-certs-standalone.sh)` | curl, openssl | Production with domain |
| Standalone (Custom IP) | `GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.../gen-certs-standalone.sh)` | curl, openssl | Production with static IP |
| Docker | `make certs` | Docker, repository | Repository-based deployment |
| Manual | `openssl commands` | openssl binary | Custom certificate needs |

### Certificate Security Levels

| Configuration | Security Level | Use Case | Alt Names |
|---------------|----------------|----------|-----------|
| Default | Development | Local testing, development | localhost, *.localhost, 127.0.0.1, ::1, 0.0.0.0 |
| Custom Domain | Production | Public servers with domain | specified-domain.com, *.specified-domain.com |
| Custom IP | Production | Servers with static IP | specified-ip-only |

**Important**: Custom certificates only include the specified domain/IP for enhanced security.
