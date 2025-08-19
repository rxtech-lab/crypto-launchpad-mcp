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
- Uniswap deployment tracking (factory, router, WETH contracts)
- Liquidity positions and swap transaction history
- Transaction sessions for signing interface management (deploy, deploy_uniswap, balance_query, pool operations)

### MCP Tools (18 total)

#### Chain Management (2 tools)

- `select_chain` - Select active blockchain (ethereum/solana)
- `set_chain` - Configure blockchain RPC and chain ID

#### Template Management (4 tools)

- `list_template` - List smart contract templates with search
- `create_template` - Create new contract template with validation
- `update_template` - Update existing template
- `delete_template` - Delete templates by ID(s) with bulk deletion support

#### Deployment (2 tools)

- `launch` - Generate deployment URL with signing interface
- `list_deployments` - List all token deployments with filtering options

#### Uniswap Integration (11 tools)

- `deploy_uniswap` - Deploy Uniswap infrastructure contracts (factory, router, WETH)
- `remove_uniswap_deployment` - Remove Uniswap deployments by ID(s) with bulk deletion support
- `set_uniswap_version` - Configure Uniswap version (v2/v3/v4)
- `get_uniswap_addresses` - Get current Uniswap configuration
- `create_liquidity_pool` - Create new liquidity pool with signing interface
- `add_liquidity` - Add liquidity to existing pool with signing interface
- `remove_liquidity` - Remove liquidity from pool with signing interface
- `swap_tokens` - Execute token swaps with signing interface
- `get_pool_info` - Retrieve pool metrics (read-only)
- `get_swap_quote` - Get swap estimates and price impact (read-only)
- `monitor_pool` - Real-time pool monitoring and event tracking (read-only)

#### Balance Query Tools (1 tool)

- `query_balance` - Query wallet balance for native tokens and ERC-20 tokens with browser/direct modes

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
make generate    # Generate embedded contract files from OpenZeppelin submodule
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
│   ├── assets/                 # Embedded HTML templates and JavaScript assets
│   └── contracts/              # OpenZeppelin contracts submodule and generated embeds
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
- ✅ **MCP Server**: 17 tools registered and functional
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
- **Modular JavaScript**: Split into focused, maintainable scripts
- **Embedded Assets**: HTML templates and JavaScript assets embedded using Go's embed directive

### JavaScript Architecture

The frontend uses a modular JavaScript architecture with separated concerns:

#### Core Scripts:

1. **`wallet-connection.js`** - Shared wallet management

   - EIP-6963 wallet discovery and connection
   - Network switching and transaction signing
   - Connection status management
   - Used by all transaction interfaces

2. **`deploy-tokens.js`** - Token deployment specific

   - Token deployment session handling
   - Transaction preparation for contract deployment
   - Success state management for contract addresses

3. **`deploy-uniswap.js`** - Uniswap deployment specific

   - Multi-contract deployment handling (WETH9, Factory, Router)
   - Uniswap-specific UI updates and progress tracking
   - Mock deployment with actual contract structure

4. **`balance-query.js`** - Balance query specific
   - Wallet balance fetching for native and ERC-20 tokens
   - Direct API calls to balance endpoints
   - Balance display updates

#### JavaScript Integration:

```html
<!-- Token Deployment -->
<script src="/js/wallet-connection.js"></script>
<script src="/js/deploy-tokens.js"></script>

<!-- Uniswap Deployment -->
<script src="/js/wallet-connection.js"></script>
<script src="/js/deploy-uniswap.js"></script>

<!-- Balance Queries -->
<script src="/js/wallet-connection.js"></script>
<script src="/js/balance-query.js"></script>
```

#### Benefits:

- **Maintainability**: Each script handles specific functionality
- **Debugging**: Easier to isolate issues to specific features
- **Performance**: Only load required JavaScript for each page
- **Reusability**: Shared wallet connection logic across all tools

### Tool Implementation Pattern

All tools follow the exact structure from the example project:

- Package `tools`
- Function signature: `func NewXxxTool(db *database.Database, ...params) (mcp.Tool, server.ToolHandlerFunc)`
- Parameter validation with required/optional parameters
- Database operations with error handling
- JSON response formatting

### Asset Management

- **Embedded Templates**: HTML templates stored in `internal/assets/` and embedded at compile time
- **Template Engine**: Go's `html/template` package for dynamic content rendering with JSON support
- **Embedded Transaction Data**: For improved performance, transaction data (including compiled bytecode) is embedded directly in HTML during template rendering instead of requiring separate API calls
- **Template Functions**: Custom template functions available:
  - `json`: Converts Go data structures to JSON for embedding in HTML data attributes
- **Modular JavaScript**: Multiple focused scripts served via HTTP endpoints:
  - `/js/wallet-connection.js` - Core wallet functionality
  - `/js/deploy-tokens.js` - Token deployment with embedded data support
  - `/js/deploy-uniswap.js` - Uniswap deployment with embedded data support
  - `/js/balance-query.js` - Balance queries
  - `/js/wallet.js` - Legacy monolithic script (deprecated)
- **Build-time Inclusion**: All assets compiled into the binary for single-file distribution

#### Embedded Data Pattern

For optimal performance, transaction data is compiled and embedded during HTML template rendering:

**Backend (Template Rendering)**:

```go
// deployment_handlers.go & uniswap_handlers.go
transactionData := s.generateTransactionData(deployment, template, activeChain)
html := s.renderTemplate("deploy", map[string]interface{}{
    "SessionID":       session.ID,
    "TransactionData": transactionData, // Embedded with bytecode
})
```

**Frontend (HTML Template)**:

```html
<div
  id="session-data"
  data-session-id="{{.SessionID}}"
  data-api-url="/api/deploy/{{.SessionID}}"
  {{if
  .TransactionData}}data-transaction-data="{{.TransactionData | json}}"
  {{end}}
></div>
```

**JavaScript (Data Loading)**:

```javascript
// Check embedded data first, fallback to API
async loadSessionData(sessionId, apiUrl, embeddedData = null) {
    if (embeddedData) {
        console.log("Using embedded transaction data");
        this.sessionData = embeddedData;
        this.displayTransactionDetails();
        return;
    }
    // Fallback to API call...
}
```

**Benefits**:

- ⚡ **Performance**: Eliminates extra API calls for transaction data
- 🔧 **Reliability**: Prevents JavaScript errors with defensive null/undefined checking
- 🔄 **Compatibility**: Maintains fallback to API calls for backward compatibility
- 🚀 **User Experience**: Faster page loads with immediate data availability

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

### Testing Architecture

All tests must use real infrastructure and follow production-like patterns:

- **Real Database Connections**: Use SQLite databases, never mocks or in-memory databases
- **Live Blockchain Testing**: Use Makefile testnet (`make e2e-network`) for all blockchain interactions
- **Production-Like Environment**: Test with actual HTTP servers, real ports, and complete request/response cycles

### Test Categories

#### 1. E2E API Tests (`/e2e/`)

Test complete API workflows with real blockchain integration:

```bash
# Start local testnet (required for all blockchain tests)
make e2e-network  # Starts anvil on localhost:8545

# Run specific API tests (30s timeout enforced)
go test -v ./e2e/api -run TestUniswapDeploymentChromedp -timeout 30s
go test -v ./e2e/api -run TestTokenDeployment -timeout 30s
go test -v ./e2e -run TestAPIServer -timeout 30s

# Run token deployment tests
go test -v ./e2e/api -run TestTokenDeploymentPageLoad -timeout 30s
go test -v ./e2e/api -run TestTokenDeploymentErrorHandling -timeout 30s
go test -v ./e2e/api -run TestTokenDeploymentWithoutWallet -timeout 30s
```

**Key Requirements:**

- Use `NewTestSetup(t)` for consistent test environment
- Verify Ethereum connection with `setup.VerifyEthereumConnection()`
- Deploy real contracts using `setup.DeployContract()`
- Test complete request/response cycles including HTML pages and JSON APIs
- Verify database updates and blockchain transaction confirmation

#### 2. Unit Tests (`/tests/`, `/internal/tools/`)

Test individual components with real dependencies:

```bash
# Run all unit tests (30s timeout enforced)
go test -v ./... -timeout 30s

# Run specific component tests
go test -v ./internal/tools -run TestUniswapUtilities -timeout 30s
go test -v ./tests -run TestUniswapDatabaseIntegration -timeout 30s
```

**Key Requirements:**

- Use temporary SQLite databases (`t.TempDir()` + `database.NewDatabase()`)
- Test actual Solidity compilation with `utils.CompileSolidity()`
- Validate real contract ABI generation and bytecode compilation
- Test database migrations and CRUD operations with real GORM

#### 3. Integration Tests (`/e2e/`)

Test cross-component interactions:

```bash
# Test complete deployment workflows (30s timeout enforced)
go test -v ./e2e -run TestAPIServer_TemplateWorkflow -timeout 30s
go test -v ./e2e -run TestContractDeployment -timeout 30s
```

### Required Test Infrastructure

#### Database Testing

```go
// E2E tests use in-memory databases for speed and isolation
db, err := database.NewDatabase(":memory:")
require.NoError(t, err)
defer db.Close()

// Unit tests can use temporary file databases when needed
tempDir := t.TempDir()
dbPath := filepath.Join(tempDir, "test.db")
db, err := database.NewDatabase(dbPath)
require.NoError(t, err)
defer db.Close()

// Test with real migrations and constraints
err = db.CreateTemplate(template)
require.NoError(t, err)
```

#### Blockchain Testing

```go
// Verify testnet connectivity
err := setup.VerifyEthereumConnection()
require.NoError(t, err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")

// Deploy real contracts
result, err := setup.DeployContract(account, contractCode, "ContractName", constructorArgs...)
require.NoError(t, err)

// Verify on-chain transaction success
receipt, err := setup.WaitForTransaction(result.TransactionHash, 30*time.Second)
require.NoError(t, err)
assert.Equal(t, uint64(1), receipt.Status)
```

#### API Testing

```go
// Test complete HTTP workflows
setup := NewTestSetup(t)
defer setup.Cleanup()

// Test HTML pages
resp, err := setup.MakeAPIRequest("GET", "/deploy-uniswap/session-id")
require.NoError(t, err)
assert.Equal(t, http.StatusOK, resp.StatusCode)
assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

// Test JSON APIs
resp, err := setup.MakeAPIRequest("GET", "/api/deploy-uniswap/session-id")
require.NoError(t, err)
assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

var apiResponse map[string]interface{}
json.NewDecoder(resp.Body).Decode(&apiResponse)
assert.Equal(t, "deploy_uniswap", apiResponse["session_type"])
```

### Test Development Commands

```bash
# Start testnet (keep running during development)
make e2e-network

# Run tests with verbose output (30s timeout enforced)
go test -v ./e2e/api -run TestUniswapDeploymentChromedp -timeout 30s
go test -v ./e2e/api -run TestTokenDeployment -timeout 30s

# Run all tests (30s timeout enforced)
make test

# Run tests with coverage (30s timeout enforced)
go test -v -cover ./... -timeout 30s

# Run specific test categories (30s timeout enforced)
go test -v ./e2e/api -timeout 30s      # Browser-based E2E API tests
go test -v ./e2e -timeout 30s          # General integration tests
go test -v ./tests -timeout 30s        # Unit tests
go test -v ./internal/... -timeout 30s # Component tests

# Token deployment specific tests
go test -v ./e2e/api -run TestTokenDeployment$ -timeout 30s           # Full deployment workflows
go test -v ./e2e/api -run TestTokenDeploymentPageLoad -timeout 30s    # Page loading tests
go test -v ./e2e/api -run TestTokenDeploymentErrorHandling -timeout 30s # Error scenarios
go test -v ./e2e/api -run TestTokenDeploymentWithoutWallet -timeout 30s # Wallet edge cases

# Uniswap deployment specific tests
go test -v ./e2e/api -run TestUniswapDeploymentChromedp -timeout 30s   # Full Uniswap workflow
go test -v ./e2e/api -run TestUniswapDeploymentPageLoad -timeout 30s   # Uniswap page tests
go test -v ./e2e/api -run TestUniswapDeploymentErrorHandling -timeout 30s # Uniswap errors
go test -v ./e2e/api -run TestUniswapDeploymentWithoutWallet -timeout 30s # Uniswap wallet tests
```

### Test Patterns and Best Practices

#### Session-Based Testing

```go
// Create real transaction sessions
sessionID, err := setup.DB.CreateTransactionSession(
    "deploy_uniswap",
    "ethereum",
    TESTNET_CHAIN_ID,
    string(sessionDataJSON),
)

// Test session lifecycle
session, err := setup.DB.GetTransactionSession(sessionID)
assert.Equal(t, "pending", session.Status)

// Test session updates
err = setup.DB.UpdateTransactionSessionStatus(sessionID, models.TransactionStatusmodels.TransactionStatusConfirmed, txHash)
assert.Equal(t, models.TransactionStatusConfirmed, session.Status)
```

#### Contract Deployment Testing

```go
// Use real contracts for testing APIs
wethResult, err := setup.DeployContract(
    account,
    GetSimpleERC20Contract(), // Real Solidity contract
    "SimpleERC20",
    "Wrapped ETH", "WETH", big.NewInt(0),
)

// Verify addresses are properly stored
updatedDeployment, err := setup.DB.GetUniswapDeploymentByID(deploymentID)
assert.Equal(t, wethResult.ContractAddress.Hex(), updatedDeployment.WETHAddress)
assert.Equal(t, wethResult.TransactionHash.Hex(), updatedDeployment.WETHTxHash)
```

#### Error Handling Testing

```go
// Test API error responses
resp, err := setup.MakeAPIRequest("GET", "/api/deploy-uniswap/invalid-session")
assert.Equal(t, http.StatusNotFound, resp.StatusCode)

var errorResponse map[string]string
json.NewDecoder(resp.Body).Decode(&errorResponse)
assert.Contains(t, errorResponse["error"], "Session not found")
```

### Continuous Integration

Tests run automatically in CI with the same infrastructure:

- **GitHub Actions**: Automated testing on push/PR
- **Anvil Testnet**: Ephemeral blockchain for each test run
- **Real Databases**: SQLite files in temporary directories
- **Complete Workflows**: End-to-end API and transaction testing

### Testing Documentation

Each test file should include:

```go
// TestUniswapDeploymentAPI tests the complete Uniswap deployment workflow
// Requirements:
// - Anvil testnet running on localhost:8545 (run 'make e2e-network')
// - Real SQLite database with migrations
// - Complete HTTP request/response testing
// - Blockchain transaction verification
func TestUniswapDeploymentAPI(t *testing.T) {
    setup := NewTestSetup(t)
    defer setup.Cleanup()

    // Verify infrastructure
    err := setup.VerifyEthereumConnection()
    require.NoError(t, err, "Testnet required: run 'make e2e-network'")

    // Test implementation...
}
```

Write comprehensive tests for:

- MCP tool implementations and parameter validation
- Database operations and model relationships
- Transaction session management
- API endpoints and error handling
- Frontend wallet integration (manual testing)
- Cross-platform binary compatibility
- CI/CD pipeline validation

### Token Deployment Test Architecture

The token deployment E2E tests are located at `/e2e/api/token_deployment_test.go` and follow the same robust patterns as Uniswap deployment tests:

#### Test Components

1. **Test Suites**:

   - `TokenDeploymentTestSuite` - Main deployment workflows (OpenZeppelin & Custom)
   - `TokenDeploymentErrorTestSuite` - Error scenarios (invalid/expired sessions)
   - `TokenDeploymentWalletTestSuite` - Wallet interaction edge cases
   - `TokenDeploymentPageLoadTestSuite` - UI functionality tests

2. **Page Object Model**: `/e2e/api/token_deployment_page.go`

   - `TokenDeploymentPage` - Encapsulates all page interactions
   - Methods for wallet selection, connection, transaction signing
   - Screenshot and debugging utilities

3. **Test Helpers**: Enhanced `/e2e/api/test_helpers.go`
   - `CreateOpenZeppelinTemplate()` - Creates ERC20 template using OpenZeppelin
   - `CreateCustomTemplate()` - Creates custom ERC20 implementation
   - `CreateTokenDeploymentSession()` - Proper session creation with deployment records

#### Test Coverage

- **Template Types**: Both OpenZeppelin-based and custom ERC20 contracts
- **Full Workflow**: Page load → Wallet connection → Transaction signing → Blockchain verification → Database updates
- **Error Handling**: Invalid sessions, expired sessions, missing wallets
- **Contract Verification**: On-chain verification of deployed contracts
- **Database Integration**: Proper session and deployment record management

#### Key Test Requirements

- Use real Anvil testnet (`make e2e-network`)
- Test actual contract compilation and deployment
- Verify bytecode generation and transaction data
- Confirm database state updates
- Browser automation with Chrome/Chromium
- 30-second timeout enforcement for all tests

## OpenZeppelin Contracts Integration

The project uses OpenZeppelin contracts as a git submodule located at `internal/contracts/openzeppelin-contracts/`.

### Git Submodule Setup

```bash
# Initialize and update the submodule
git submodule init
git submodule update

# Or clone with submodules
git clone --recurse-submodules <repository-url>
```

### Embedding Contracts

The OpenZeppelin contracts are embedded in the Go binary using Go's embed directive. The embedding is automated through code generation:

1. **Generate embed directives**: Run `make generate` to scan all `.sol` files in the OpenZeppelin contracts submodule and generate the appropriate `//go:embed` directives in `internal/contracts/contracts.go`.

2. **Code Generation**: The `tools/generate_contracts.go` script automatically creates embed directives for all Solidity files found in the submodule, ensuring all contracts are available at compile time.

3. **Build Integration**: The generation step should be run whenever the OpenZeppelin submodule is updated or when setting up the project for the first time.

### Usage in Solidity Compilation

The `utils/solidity.go` file includes an import callback for `@openzeppelin-contracts/` imports that resolves to the embedded filesystem, enabling seamless compilation of contracts that depend on OpenZeppelin libraries.

## Key Dependencies

- `github.com/mark3labs/mcp-go` - MCP server framework
- `github.com/gofiber/fiber/v2` - HTTP framework with middleware
- `gorm.io/gorm` and `gorm.io/driver/sqlite` - ORM and database driver
- `github.com/google/uuid` - UUID generation for sessions
- `github.com/rxtech-lab/solc-go` - Solidity compiler for contract validation
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

# Code guideline

1. Never use fmt.Println to log something
2. **Test Timeout Policy**: Never run tests longer than 30 seconds. Use `-timeout 30s` for all test commands to enforce this limit and prevent hanging tests.

## E2E Testing Best Practices

### HTML Testing Guidelines

When testing HTML responses in E2E tests:

1. **Focus on Functionality, Not Implementation Details**

   - Test for essential data elements using IDs: `id="session-data"`
   - Verify embedded transaction data: `data-transaction-data=`
   - Check for session IDs and critical data attributes
   - DO NOT test for specific JavaScript files or CSS classes
   - Test files are located in /e2e/page

2. **Use Element IDs for Reliable Testing**

   ```go
   // Good - tests functionality
   s.Assert().Contains(htmlContent, `id="session-data"`)
   s.Assert().Contains(htmlContent, "data-transaction-data=")

   // Bad - tests implementation details
   s.Assert().Contains(htmlContent, "wallet-connection.js")
   s.Assert().Contains(htmlContent, "class=\"bg-gray-100\"")
   ```

3. **API Response Testing**

   - Test the actual structure returned by handlers
   - Don't assume all transaction data is included in generic API responses
   - Verify critical fields based on what the handler actually returns

4. **Database State Verification**
   - Always verify database state changes after operations
   - Check transaction hashes are properly stored
   - Verify status updates are applied correctly
