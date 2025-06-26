#!/bin/bash

# GrpcHub Standalone Certificate Generation Script
# This script generates TLS certificates for GrpcHub without requiring repository clone
# Usage: curl -sSL https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/gen-certs-standalone.sh | bash
# Custom domain/IP: GRPCHUB_DOMAIN=example.com bash <(curl -sSL https://...)
# Custom IP: GRPCHUB_IP=192.168.1.100 bash <(curl -sSL https://...)

set -e

genpkcs8key() {
    local key_name="$1"
    local key_bits="$2"
    local temp_key="${key_name}_rsa.key"
    local final_key="${key_name}.key"
    
    # 生成 RSA 密钥
    openssl genrsa -out "$temp_key" "$key_bits"
    
    # 检查是否已经是 PKCS#8 格式
    if head -1 "$temp_key" | grep -q "BEGIN PRIVATE KEY"; then
        # 已经是 PKCS#8 格式，直接重命名
        mv "$temp_key" "$final_key"
    else
        # 不是 PKCS#8 格式，需要转换
        openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in "$temp_key" -out "$final_key"
        rm -f "$temp_key"
    fi
}

echo "🔐 GrpcHub Certificate Generator"
echo "================================"

# Parse command line arguments
CUSTOM_DOMAIN=""
CUSTOM_IP=""
USE_DEFAULTS=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            CUSTOM_DOMAIN="$2"
            USE_DEFAULTS=false
            shift 2
            ;;
        --ip)
            CUSTOM_IP="$2"
            USE_DEFAULTS=false
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --domain <domain>    Custom domain name"
            echo "  --ip <ip>           Custom IP address"
            echo "  -h, --help          Show this help"
            echo ""
            echo "Environment variables:"
            echo "  GRPCHUB_DOMAIN      Custom domain name"
            echo "  GRPCHUB_IP          Custom IP address"
            echo ""
            echo "Examples:"
            echo "  $0 --domain example.com"
            echo "  $0 --ip 192.168.1.100"
            echo "  GRPCHUB_DOMAIN=example.com $0"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Check environment variables if not set via arguments
if [[ -n "$GRPCHUB_DOMAIN" && -z "$CUSTOM_DOMAIN" ]]; then
    CUSTOM_DOMAIN="$GRPCHUB_DOMAIN"
    USE_DEFAULTS=false
fi

if [[ -n "$GRPCHUB_IP" && -z "$CUSTOM_IP" ]]; then
    CUSTOM_IP="$GRPCHUB_IP"
    USE_DEFAULTS=false
fi

# Validate inputs
if [[ -n "$CUSTOM_DOMAIN" ]]; then
    if [[ ! "$CUSTOM_DOMAIN" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
        echo "❌ Invalid domain name: $CUSTOM_DOMAIN"
        exit 1
    fi
    echo "🌐 Using custom domain: $CUSTOM_DOMAIN"
fi

if [[ -n "$CUSTOM_IP" ]]; then
    if [[ ! "$CUSTOM_IP" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$|^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}$ ]]; then
        echo "❌ Invalid IP address: $CUSTOM_IP"
        exit 1
    fi
    echo "🌐 Using custom IP: $CUSTOM_IP"
fi

# Create certificate directory
CERT_DIR="certs"
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# Clean up any existing certificates
rm -f *.pem *.crt *.key *.csr *.srl

echo "📋 Generating certificates..."

# Certificate details
CA_SUBJECT="/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT Department/CN=GrpcHub CA"
if [[ -n "$CUSTOM_DOMAIN" ]]; then
    SERVER_SUBJECT="/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT Department/CN=$CUSTOM_DOMAIN"
elif [[ -n "$CUSTOM_IP" ]]; then
    SERVER_SUBJECT="/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT Department/CN=grpchub-server"
else
    SERVER_SUBJECT="/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT Department/CN=localhost"
fi
CLIENT_SUBJECT="/C=US/ST=CA/L=San Francisco/O=GrpcHub/OU=IT Department/CN=grpchub-client"

echo "🔑 1. Generating CA certificate..."
# Generate CA private key
genpkcs8key "ca" 4096

# Generate CA root certificate
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -subj "$CA_SUBJECT"

echo "🖥️  2. Generating server certificate..."
# Generate server private key
genpkcs8key "server" 2048

# Generate server certificate signing request
openssl req -new -key server.key -out server.csr -subj "$SERVER_SUBJECT"

# Create server certificate extensions
if [[ "$USE_DEFAULTS" == "true" ]]; then
    # Default configuration with localhost and common IPs
    cat > server.ext << EOF
basicConstraints = CA:FALSE
keyUsage = nonRepudiation,digitalSignature,keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
DNS.3 = grpchub-server
DNS.4 = *.grpchub-server
IP.1 = 127.0.0.1
IP.2 = ::1
IP.3 = 0.0.0.0
EOF
    echo "🏠 Using default configuration (localhost, 127.0.0.1, ::1)"
else
    # Custom configuration - only use provided domain/IP for security
    cat > server.ext << EOF
basicConstraints = CA:FALSE
keyUsage = nonRepudiation,digitalSignature,keyEncipherment
subjectAltName = @alt_names

[alt_names]
EOF

    alt_count=1
    if [[ -n "$CUSTOM_DOMAIN" ]]; then
        echo "DNS.$alt_count = $CUSTOM_DOMAIN" >> server.ext
        alt_count=$((alt_count + 1))
        echo "DNS.$alt_count = *.$CUSTOM_DOMAIN" >> server.ext
        alt_count=$((alt_count + 1))
        echo "🔒 Added DNS names: $CUSTOM_DOMAIN, *.$CUSTOM_DOMAIN"
    fi

    if [[ -n "$CUSTOM_IP" ]]; then
        echo "IP.$alt_count = $CUSTOM_IP" >> server.ext
        echo "🔒 Added IP address: $CUSTOM_IP"
    fi

    echo "⚠️  Security: Certificate restricted to specified domain/IP only"
fi

# Generate server certificate
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365 -extfile server.ext

echo "👤 3. Generating client certificate..."
# Generate client private key
genpkcs8key "client" 2048

# Generate client certificate signing request
openssl req -new -key client.key -out client.csr -subj "$CLIENT_SUBJECT"

# Create client certificate extensions
cat > client.ext << EOF
basicConstraints = CA:FALSE
keyUsage = nonRepudiation,digitalSignature,keyEncipherment
extendedKeyUsage = clientAuth
EOF

# Generate client certificate
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 365 -extfile client.ext

echo "📦 4. Creating combined PEM files..."

# Create server PEM file (certificate + private key + CA)
echo "# GrpcHub Server Certificate and Private Key" > server.pem
cat server.crt >> server.pem
echo "" >> server.pem
cat server.key >> server.pem
echo "" >> server.pem
echo "# CA Root Certificate" >> server.pem
cat ca.crt >> server.pem

# Create client PEM file (certificate + private key + CA)
echo "# GrpcHub Client Certificate and Private Key" > client.pem
cat client.crt >> client.pem
echo "" >> client.pem
cat client.key >> client.pem
echo "" >> client.pem
echo "# CA Root Certificate" >> client.pem
cat ca.crt >> client.pem

echo "🧹 5. Cleaning up temporary files..."
rm -f *.csr *.srl *.ext

echo ""
echo "✅ Certificate generation completed!"
echo ""
echo "📁 Generated files:"
echo "   📄 ca.crt      - CA root certificate"
echo "   🔐 ca.key      - CA private key"
echo "   📄 server.crt  - Server certificate"
echo "   🔐 server.key  - Server private key"
echo "   📄 client.crt  - Client certificate"
echo "   🔐 client.key  - Client private key"
echo "   📦 server.pem  - Server bundle (for GrpcHub server)"
echo "   📦 client.pem  - Client bundle (for client applications)"
echo ""

if [[ "$USE_DEFAULTS" == "true" ]]; then
    echo "🏠 Certificate configuration:"
    echo "   Valid for: localhost, *.localhost, 127.0.0.1, ::1"
    echo "   Suitable for: Development and local testing"
else
    echo "🔒 Certificate configuration:"
    [[ -n "$CUSTOM_DOMAIN" ]] && echo "   Domain: $CUSTOM_DOMAIN, *.$CUSTOM_DOMAIN"
    [[ -n "$CUSTOM_IP" ]] && echo "   IP: $CUSTOM_IP"
    echo "   Security: Restricted to specified domain/IP only"
fi

echo ""
echo "🚀 Usage:"
echo "   Server: Use server.pem with --pem parameter"
echo "   Client: Use client.pem for client authentication"
echo ""
echo "🔍 Verify certificates:"
echo "   openssl x509 -in server.crt -text -noout"
echo "   openssl verify -CAfile ca.crt server.crt"
echo ""
echo "🔒 Security Notes:"
if [[ "$USE_DEFAULTS" == "true" ]]; then
    echo "   - Default certificates include localhost and common IPs"
    echo "   - Suitable for development and local testing"
    echo "   - For production, use custom domain/IP for better security"
else
    echo "   - Custom certificates are restricted to specified domain/IP only"
    echo "   - More secure for production use"
    echo "   - Ensure your server is accessible via the specified domain/IP"
fi
echo "   - Keep ca.key secure - it can sign new certificates"
echo "   - Consider using certificates from a trusted CA for production"
