# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Crypto Launchpad MCP (Model Context Protocol) server built in Go that allows AI agents to manage cryptocurrency token deployments and Uniswap liquidity operations. The project combines an MCP server for AI tool integration with a REST API for blockchain transaction signing interfaces.

## Core Architecture

- **MCP Server**: Built using `github.com/mark3labs/mcp-go` for creating MCP tools
- **REST API**: Fiber framework with random port assignment for transaction signing interfaces
- **Database**: GORM with SQLite for local data storage
- **Frontend**: HTMX + Tailwind CSS for reactive signing interfaces
- **Blockchain Integration**: EIP-6963 wallet discovery for Ethereum and Solana support

## Key Components

### Data Models
- Chain configurations (blockchain RPC endpoints and chain IDs)
- Smart contract templates by chain type (Ethereum/Solana)
- Deployment records with transaction tracking
- Uniswap settings and pool management
- Liquidity positions and swap transaction history
- Transaction sessions for signing interface management

### MCP Tools (14 total)

#### Chain Management (2 tools)
- `select_chain` - Select active blockchain (ethereum/solana)
- `set_chain` - Configure blockchain RPC and chain ID

#### Template Management (3 tools)
- `list_template` - List smart contract templates with search
- `create_template` - Create new contract template with validation
- `update_template` - Update existing template

#### Deployment (1 tool)
- `launch` - Generate deployment URL with signing interface

#### Uniswap Integration (8 tools)
- `set_uniswap_version` - Configure Uniswap version (v2/v3/v4)
- `create_liquidity_pool` - Create new liquidity pool with signing interface
- `add_liquidity` - Add liquidity to existing pool with signing interface
- `remove_liquidity` - Remove liquidity from pool with signing interface
- `swap_tokens` - Execute token swaps with signing interface
- `get_pool_info` - Retrieve pool metrics (read-only)
- `get_swap_quote` - Get swap estimates and price impact (read-only)
- `monitor_pool` - Real-time pool monitoring and event tracking (read-only)

### Transaction Signing Workflow
1. AI tool creates transaction session in database
2. Tool generates unique URL with session ID
3. User opens URL in browser
4. Frontend loads with EIP-6963 wallet discovery
5. User connects wallet and reviews transaction details
6. User signs and sends transaction
7. Frontend updates session status via API
8. Database records are updated with transaction hash

## Development Commands

All commands are defined in the Makefile with comprehensive build system:

```bash
# Basic Commands
make build       # Build the project with version info
make test        # Run tests
make run         # Run the MCP server directly (no build)
make run-bin     # Build and run the binary
make deps        # Download and tidy dependencies
make clean       # Clean build artifacts

# Distribution Commands
make binaries    # Build for multiple architectures (darwin/linux/windows)
make install-local # Install to /usr/local/bin (requires sudo)
make package     # Package and notarize for distribution (macOS only)

# Code Quality
make fmt         # Format code with gofmt
make lint        # Run golangci-lint
make sec         # Run security scan with gosec

# Version Information
./bin/launchpad-mcp --version  # Show version, commit, build time
./bin/launchpad-mcp --help     # Show help and usage
```

### Build System Features
- **Version Information**: Build flags inject version, commit hash, and build time
- **Cross-Platform Builds**: Support for darwin/linux/windows on amd64/arm64
- **Code Signing**: macOS code signing with hardened runtime (requires certificates)
- **Notarization**: Apple notarization for distribution (requires Apple ID)
- **Automated Packaging**: Creates .pkg installer for macOS distribution

## File Structure

```
├── docs/Design.md              # Detailed design specifications
├── cmd/main.go                 # Main application entry point with version support
├── internal/                   # Core business logic
│   ├── models/                 # GORM data models
│   ├── database/               # Database layer with CRUD operations
│   ├── mcp/                    # MCP server implementation
│   ├── api/                    # HTTP server for transaction signing
│   └── assets/                 # Embedded HTML templates and JavaScript assets
├── tools/                      # 14 MCP tool implementations
├── scripts/                    # Build and distribution scripts
│   ├── binaries.sh            # Cross-platform build script
│   ├── sign.sh                # macOS code signing script
│   ├── package-notarize.sh    # macOS packaging and notarization
│   └── post-install.sh        # Post-installation setup
├── .github/workflows/          # CI/CD automation
│   ├── ci.yml                 # Continuous integration
│   ├── release.yml            # Release automation
│   └── create-release.yaml    # Semantic release creation
├── .golangci.yml              # Linting configuration
├── Makefile                   # Build system commands
├── CLAUDE.md                  # Development guidance (this file)
└── README.md                  # Project documentation
```

## Implementation Status

- ✅ **Complete Implementation**: All core components implemented and ready
- ✅ **MCP Server**: 14 tools registered and functional
- ✅ **Database Layer**: GORM with SQLite, automatic migrations
- ✅ **HTTP Server**: Random port assignment, transaction signing interfaces
- ✅ **Frontend**: EIP-6963 wallet integration, HTMX + Tailwind CSS
- ✅ **Dual Server Setup**: MCP (stdio) and HTTP servers running concurrently

## Architecture Decisions

### Database Design
- **SQLite**: Local database for easy deployment and development
- **GORM**: Type-safe ORM with automatic migrations
- **Session Management**: 30-minute expiry for security

### HTTP Server Design
- **Random Port**: Uses `net.Listen("tcp", ":0")` for automatic port assignment
- **Session-based URLs**: Unique URLs for each transaction signing session
- **RESTful API**: Clean separation between page serving and API endpoints

### Frontend Design
- **EIP-6963**: Standard wallet discovery for maximum compatibility
- **Progressive Enhancement**: Works without JavaScript for basic functionality
- **Responsive Design**: Tailwind CSS for mobile-friendly interfaces
- **Embedded Assets**: HTML templates and JavaScript assets embedded using Go's embed directive

### Tool Implementation Pattern
All tools follow the exact structure from the example project:
- Package `tools`
- Function signature: `func NewXxxTool(db *database.Database, ...params) (mcp.Tool, server.ToolHandlerFunc)`
- Parameter validation with required/optional parameters
- Database operations with error handling
- JSON response formatting

### Asset Management
- **Embedded Templates**: HTML templates stored in `internal/assets/` and embedded at compile time
- **Template Engine**: Go's `text/template` package for dynamic content rendering
- **Static Assets**: JavaScript files embedded and served via HTTP endpoints
- **Build-time Inclusion**: All assets compiled into the binary for single-file distribution

## Security Considerations

- **Input Validation**: All user inputs validated before database operations
- **Template Validation**: Smart contract templates checked for basic security issues
- **Session Expiry**: Transaction sessions expire after 30 minutes
- **No Private Keys**: System never handles private keys - all signing done client-side

## Important Notes

- Database file: `~/launchpad.db` (SQLite) created automatically in user home directory
- The server runs both MCP (stdio) and HTTP (random port) simultaneously
- All blockchain operations require user wallet signatures - no server-side signing
- Uniswap operations currently support Ethereum only (v2 fully supported)
- Transaction signing requires modern web browser with EIP-6963 compatible wallet

## CI/CD and Release Process

### Continuous Integration
- **GitHub Actions**: Automated testing on push/PR to main/develop branches
- **Multi-Platform Testing**: Tests run on ubuntu-latest with Go 1.24
- **Code Quality**: Formatting checks, linting, and security scanning
- **Build Verification**: Cross-platform binary compilation

### Release Automation
- **Semantic Versioning**: Automated version bumping based on commit messages
- **Cross-Platform Builds**: Binaries for darwin/linux/windows (amd64/arm64)
- **macOS Distribution**: Code-signed and notarized .pkg installer
- **GitHub Releases**: Automatic asset uploads and release notes

### Code Signing and Notarization
Required environment variables for production builds:
```bash
# Code Signing (macOS)
SIGNING_CERTIFICATE_NAME="Developer ID Application: Your Name"
INSTALLER_SIGNING_CERTIFICATE_NAME="Developer ID Installer: Your Name"

# Notarization (Apple)
APPLE_ID="your-apple-id@example.com"
APPLE_ID_PWD="app-specific-password"
APPLE_TEAM_ID="YOUR_TEAM_ID"

# GitHub Secrets (base64 encoded)
BUILD_CERTIFICATE_BASE64="..."
INSTALLER_CERTIFICATE_BASE64="..."
P12_PASSWORD="certificate-password"
```

## Testing Strategy

Write comprehensive tests for:
- MCP tool implementations and parameter validation
- Database operations and model relationships
- Transaction session management
- API endpoints and error handling
- Frontend wallet integration (manual testing)
- Cross-platform binary compatibility
- CI/CD pipeline validation

## Key Dependencies

- `github.com/mark3labs/mcp-go` - MCP server framework
- `github.com/gofiber/fiber/v2` - HTTP framework with middleware
- `gorm.io/gorm` and `gorm.io/driver/sqlite` - ORM and database driver
- `github.com/google/uuid` - UUID generation for sessions
- Tailwind CSS + HTMX - Frontend framework (CDN)

## Distribution and Installation

### Development Installation
```bash
# Clone and build locally
git clone <repository-url>
cd launchpad-mcp
make deps
make build
make install-local  # Installs to /usr/local/bin
```

### Production Distribution
- **macOS**: Download signed .pkg installer from GitHub releases
- **Linux/Windows**: Download appropriate binary from GitHub releases
- **Manual Install**: Use `make install-local` after building from source

### Claude Desktop Integration
Add to Claude Desktop MCP configuration:
```json
{
  "launchpad-mcp": {
    "command": "/usr/local/bin/launchpad-mcp",
    "args": []
  }
}
```

## Extension Points

The architecture supports easy extension for:
- Additional blockchain networks (Polygon, BSC, etc.)
- More Uniswap versions (v3, v4 with concentrated liquidity)
- Additional DEX integrations (SushiSwap, PancakeSwap)
- Advanced trading features (limit orders, DCA)
- Portfolio tracking and analytics
- Multi-sig wallet support
- Custom deployment templates and validation rules
- Real-time price feeds and market data integration