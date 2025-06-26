# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Rust Edition**: Updated from 2021 to 2024
- **Go Version**: Updated minimum requirement to Go 1.24.2
- **Module Management**: Simplified Go module dependencies using replace directives instead of go.work
- **Documentation**: Updated all documentation to reflect current Go version requirements and module configuration

### Updated Dependencies
- Various dependency versions updated to latest compatible versions
- Maintained compatibility with existing APIs

### Documentation Updates
- Updated `README.md` to reflect Go 1.24.2 requirement
- Updated `examples/go/README.md` to correct module configuration documentation
- Updated `deploy/README.md` to include system requirements
- Corrected workspace configuration documentation (removed go.work references, documented replace directive usage)

### Technical Changes
- Rust edition 2024 provides improved language features and performance
- Go 1.24.2 requirement ensures access to latest language features and security updates
- Simplified module management reduces complexity for developers

## [0.1.0] - Initial Release

### Added
- High-performance gRPC message relay server
- Bidirectional streaming communication
- TLS security with mutual authentication
- Channel management and client registration
- Health monitoring and service reflection
- Zstd compression support
- Multi-language support (Rust server, Go client SDK)
- Docker deployment configurations
- Certificate generation scripts
- Comprehensive documentation and examples

### Features
- Real-time message relay between clients
- Secure TLS channels with certificate validation
- Dynamic client registration and connection handling
- Built-in health checks and service reflection
- Optimized performance with compression
- Production-ready Docker deployment
- Easy certificate management
- Complete Go client library with examples