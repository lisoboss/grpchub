#!/bin/bash

# GrpcHub One-liner Installer
# Similar to Rust installer: curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
# Usage: curl --proto '=https' --tlsv1.2 -sSf https://raw.githubusercontent.com/lisoboss/grpchub/main/scripts/install.sh | sh

set -e

GRPCHUB_REPO="https://raw.githubusercontent.com/lisoboss/grpchub/main"
INSTALL_DIR="${GRPCHUB_HOME:-$HOME/.grpchub}"
COMPOSE_FILE="docker-compose.ghcr.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}info${NC}: $1"
}

print_success() {
    echo -e "${GREEN}success${NC}: $1"
}

print_warning() {
    echo -e "${YELLOW}warning${NC}: $1"
}

print_error() {
    echo -e "${RED}error${NC}: $1"
    exit 1
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command_exists docker; then
        print_error "Docker is required but not installed. Please install Docker first."
    fi
    
    if ! command_exists docker-compose; then
        print_error "Docker Compose is required but not installed. Please install Docker Compose first."
    fi
    
    if ! command_exists curl; then
        print_error "curl is required but not installed. Please install curl first."
    fi
    
    if ! command_exists openssl; then
        print_error "OpenSSL is required but not installed. Please install OpenSSL first."
    fi
    
    print_success "All prerequisites are satisfied"
}

# Create installation directory
create_install_dir() {
    print_info "Creating installation directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"
}

# Download required files
download_files() {
    print_info "Downloading GrpcHub deployment files..."
    
    # Download docker-compose.ghcr.yaml
    curl -sSf "$GRPCHUB_REPO/deploy/$COMPOSE_FILE" -o "$COMPOSE_FILE" || {
        print_error "Failed to download $COMPOSE_FILE"
    }
    
    # Download certificate generation script
    curl -sSf "$GRPCHUB_REPO/scripts/gen-certs-standalone.sh" -o "gen-certs.sh" || {
        print_error "Failed to download certificate generation script"
    }
    
    chmod +x gen-certs.sh
    
    print_success "Downloaded deployment files"
}

# Generate certificates
generate_certificates() {
    print_info "Generating TLS certificates..."
    
    if [ -f "certs/server.pem" ]; then
        print_warning "Certificates already exist, skipping generation"
        return
    fi
    
    # Set certificate generation options
    local cert_args=""
    if [[ -n "$GRPCHUB_DOMAIN" ]]; then
        cert_args="$cert_args --domain $GRPCHUB_DOMAIN"
        print_info "Using custom domain: $GRPCHUB_DOMAIN"
    fi
    
    if [[ -n "$GRPCHUB_IP" ]]; then
        cert_args="$cert_args --ip $GRPCHUB_IP"
        print_info "Using custom IP: $GRPCHUB_IP"
    fi
    
    if [[ -z "$cert_args" ]]; then
        print_info "Using default certificate configuration (localhost, 127.0.0.1)"
    fi
    
    ./gen-certs.sh $cert_args || {
        print_error "Failed to generate certificates"
    }
    
    print_success "Generated TLS certificates"
}

# Pull Docker image
pull_image() {
    print_info "Pulling GrpcHub Docker image..."
    
    docker pull ghcr.io/lisoboss/grpchub:latest || {
        print_error "Failed to pull Docker image"
    }
    
    print_success "Pulled Docker image"
}

# Create convenience scripts
create_scripts() {
    print_info "Creating convenience scripts..."
    
    # Create start script
    cat > start.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
docker-compose -f docker-compose.ghcr.yaml up -d
echo "GrpcHub started successfully!"
echo "Server is available at: localhost:50055"
echo "View logs with: ./logs.sh"
EOF
    
    # Create stop script
    cat > stop.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
docker-compose -f docker-compose.ghcr.yaml down
echo "GrpcHub stopped successfully!"
EOF
    
    # Create logs script
    cat > logs.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
docker-compose -f docker-compose.ghcr.yaml logs -f
EOF
    
    # Create status script
    cat > status.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
docker-compose -f docker-compose.ghcr.yaml ps
EOF
    
    # Create update script
    cat > update.sh << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
echo "Updating GrpcHub..."
docker pull ghcr.io/lisoboss/grpchub:latest
docker-compose -f docker-compose.ghcr.yaml up -d
echo "GrpcHub updated successfully!"
EOF
    
    # Make scripts executable
    chmod +x start.sh stop.sh logs.sh status.sh update.sh
    
    print_success "Created convenience scripts"
}

# Add to PATH (optional)
setup_path() {
    local shell_profile=""
    
    if [ -n "$BASH_VERSION" ]; then
        shell_profile="$HOME/.bashrc"
    elif [ -n "$ZSH_VERSION" ]; then
        shell_profile="$HOME/.zshrc"
    elif [ -f "$HOME/.profile" ]; then
        shell_profile="$HOME/.profile"
    fi
    
    if [ -n "$shell_profile" ] && [ -f "$shell_profile" ]; then
        if ! grep -q "GRPCHUB_HOME" "$shell_profile"; then
            echo "" >> "$shell_profile"
            echo "# GrpcHub" >> "$shell_profile"
            echo "export GRPCHUB_HOME=\"$INSTALL_DIR\"" >> "$shell_profile"
            echo "export PATH=\"\$GRPCHUB_HOME:\$PATH\"" >> "$shell_profile"
            print_info "Added GrpcHub to PATH in $shell_profile"
        fi
    fi
}

# Start GrpcHub
start_grpchub() {
    print_info "Starting GrpcHub..."
    
    docker-compose -f "$COMPOSE_FILE" up -d || {
        print_error "Failed to start GrpcHub"
    }
    
    print_success "GrpcHub started successfully!"
}

# Print final instructions
print_instructions() {
    echo ""
    echo "🎉 GrpcHub installation completed successfully!"
    echo ""
    echo "📁 Installation directory: $INSTALL_DIR"
    echo "📡 Server is running at: localhost:50055"
    echo ""
    echo "🚀 Quick commands:"
    echo "   Start:  $INSTALL_DIR/start.sh"
    echo "   Stop:   $INSTALL_DIR/stop.sh"
    echo "   Logs:   $INSTALL_DIR/logs.sh"
    echo "   Status: $INSTALL_DIR/status.sh"
    echo "   Update: $INSTALL_DIR/update.sh"
    echo ""
    echo "📋 Certificate files are located in: $INSTALL_DIR/certs/"
    echo "   server.pem - Use this for server configuration"
    echo "   client.pem - Use this for client applications"
    echo ""
    if [[ -n "$GRPCHUB_DOMAIN" || -n "$GRPCHUB_IP" ]]; then
        echo "🔒 Security: Custom certificates generated"
        [[ -n "$GRPCHUB_DOMAIN" ]] && echo "   Domain: $GRPCHUB_DOMAIN"
        [[ -n "$GRPCHUB_IP" ]] && echo "   IP: $GRPCHUB_IP"
        echo "   Certificates are restricted to specified domain/IP only"
    else
        echo "🏠 Default certificates for localhost and local IPs"
        echo "   For production, consider using --domain or --ip options"
    fi
    echo ""
    echo "🔧 Configuration:"
    echo "   Edit $INSTALL_DIR/.env to customize settings"
    echo ""
    echo "📚 Documentation:"
    echo "   https://github.com/lisoboss/grpchub-go"
    echo ""
    print_success "Happy gRPC messaging! 🚀"
}

# Main installation process
main() {
    echo "GrpcHub Installer"
    echo "=================="
    echo ""
    
    check_prerequisites
    create_install_dir
    download_files
    generate_certificates
    pull_image
    create_scripts
    setup_path
    start_grpchub
    print_instructions
}

# Handle script arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "GrpcHub Installer"
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --help, -h           Show this help message"
            echo "  --no-start           Don't start GrpcHub after installation"
            echo "  --dir <directory>    Install to custom directory"
            echo "  --domain <domain>    Custom domain for TLS certificate"
            echo "  --ip <ip>           Custom IP for TLS certificate"
            echo ""
            echo "Environment variables:"
            echo "  GRPCHUB_HOME         Custom installation directory"
            echo "  GRPCHUB_DOMAIN       Custom domain for TLS certificate"
            echo "  GRPCHUB_IP           Custom IP for TLS certificate"
            echo ""
            echo "Examples:"
            echo "  $0 --domain example.com"
            echo "  $0 --ip 192.168.1.100"
            echo "  GRPCHUB_DOMAIN=example.com $0"
            echo ""
            exit 0
            ;;
        --no-start)
            NO_START=1
            shift
            ;;
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --domain)
            GRPCHUB_DOMAIN="$2"
            shift 2
            ;;
        --ip)
            GRPCHUB_IP="$2"
            shift 2
            ;;
        *)
            print_error "Unknown option: $1"
            ;;
    esac
done

# Run main installation if not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ "$NO_START" == "1" ]]; then
        # Override start function to do nothing
        start_grpchub() {
            print_info "Skipping GrpcHub startup (--no-start specified)"
        }
    fi
    
    main "$@"
fi