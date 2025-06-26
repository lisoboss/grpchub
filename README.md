# GrpcHub

A high-performance gRPC message relay server that enables bidirectional streaming communication between clients through secure channels.

## Features

- **Bidirectional Streaming**: Real-time message relay between clients
- **TLS Security**: Mutual TLS authentication with certificate validation
- **Channel Management**: Dynamic client registration and connection handling
- **Health Monitoring**: Built-in health check and service reflection
- **Compression**: Zstd compression for optimized performance
- **Multi-language Support**: Rust server with Go client SDK

## Quick Start

### Docker (Recommended)

#### Option 1: One-liner Install (Rust-style)

```bash
# Install and start GrpcHub (similar to Rust installer)
curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/install.sh | sh
```

#### Option 2: Direct Download (Manual)

```bash
# Download the deployment configuration
curl -O https://raw.githubusercontent.com/lisoboss/grpchub/main/deploy/docker-compose.ghcr.yaml

# Generate TLS certificates (one-liner)
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash

# Or with custom domain/IP for production:
# GRPCHUB_DOMAIN=your-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)
# GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Start the server
docker-compose -f docker-compose.ghcr.yaml up -d
```

#### Option 3: Clone Repository (Full Development)

```bash
# Clone the repository
git clone https://github.com/lisoboss/grpchub.git
cd grpchub/deploy

# Quick start with Docker (build from source)
make quick-start

# Or use pre-built image from GitHub Container Registry
make quick-start-ghcr
```

### Manual Build

```bash
# Build and run the server
cargo run --bin grpchub-serve -- --addr "[::1]:50055" --pem ./server.pem
```

### Client (Go)

This project requires **Go 1.24.2** or later and uses local module dependencies with replace directives. From the project root:

```bash
# Run examples
go run ./examples/go server
go run ./examples/go client
```

For complete Go client and server examples, see the [examples/go](examples/go) directory.

The examples include:
- TLS certificate loading from PEM files
- Creating GrpcHub clients and servers
- Proper connection management
- Standard gRPC usage patterns

**Key concepts:**
- Each client and server needs a unique component ID
- Use `client.pem` (not `server.pem`) for Go applications
- Standard gRPC patterns work through the `grpcx` package
- Local module dependencies managed via replace directives in go.mod

### Certificate Files

After generating certificates, you'll have:
- `server.pem` - Used by the GrpcHub server (--pem parameter)
- `client.pem` - Used by Go clients and servers connecting to GrpcHub
- `ca.crt` - Certificate Authority (for verification)

## Architecture

- **Server**: Rust-based gRPC server with tokio async runtime
- **Protocol**: Protocol Buffers v3 with bidirectional streaming
- **Transport**: HTTP/2 with TLS 1.3 encryption
- **Client SDK**: Go library with Kratos framework integration

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--addr` | Server listen address | `[::1]:50055` |
| `--pem` | TLS certificate file path | `./server.pem` |

## Message Types

- `PT_HELLO`: Connection handshake
- `PT_HEADER`: Metadata transmission
- `PT_PAYLOAD`: Message content
- `PT_CLOSE`: Connection termination
- `PT_ERROR`: Error handling

## Deployment

For production deployment, see the [deployment guide](deploy/README.md).

### Docker Compose

```bash
cd deploy

# Production deployment (build from source)
make run

# Production deployment (using GHCR image)
make ghcr

# Development environment
make dev

# View logs
make logs
```

### Configuration Options

```bash
# For direct download deployment
# Download environment template (optional)
curl -O https://raw.githubusercontent.com/lisoboss/grpchub/main/deploy/.env.example
# Edit .env file with your settings

# For cloned repository
cp deploy/.env.example deploy/.env
# Edit .env file with your settings

# Using Make commands (cloned repository only)
make certs              # Generate TLS certificates
make health             # Check service health
make quick-start-ghcr   # Quick start with GHCR image
make clean              # Clean up containers
```

## Certificate Generation

### Quick Certificate Generation (Standalone)

For production deployment without cloning the repository:

```bash
# Default certificates (localhost, 127.0.0.1, ::1)
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash

# Custom domain (production recommended)
GRPCHUB_DOMAIN=your-domain.com bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Custom IP address
GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh)

# Using command line arguments
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash -s -- --domain your-domain.com
curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash -s -- --ip 192.168.1.100
```

This generates:
- `server.pem` - Server certificate and private key
- `client.pem` - Client certificate and private key
- `ca.crt` - Certificate Authority

**Security Note**: Custom domain/IP certificates are restricted to only the specified values for enhanced security.

### Manual Certificate Generation

```bash
# Create certificate directory
mkdir -p certs && cd certs

# Generate CA private key
openssl genrsa -out ca.key 4096

# Generate CA certificate
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -subj "/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT/CN=GrpcHub CA"

# Generate server private key
openssl genrsa -out server.key 2048

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT/CN=localhost"

# Generate server certificate
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extensions v3_req -extfile <(echo -e "basicConstraints=CA:FALSE\nkeyUsage=nonRepudiation,digitalSignature,keyEncipherment\nsubjectAltName=@alt_names\n[alt_names]\nDNS.1=localhost\nDNS.2=*.localhost\nIP.1=127.0.0.1\nIP.2=::1")

# Create combined PEM file
cat server.crt server.key ca.crt > server.pem

# Generate client certificate (similar process)
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT/CN=client"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365
cat client.crt client.key ca.crt > client.pem

# Cleanup
rm *.csr *.srl
```

## Development

```bash
# Generate protobuf code
buf generate

# Run tests
cargo test

# Development with Docker
cd deploy && make dev

# Manual development
cargo run --bin grpchub-serve
```
