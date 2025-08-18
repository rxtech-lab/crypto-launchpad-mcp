# E2E Tests for Launchpad MCP API Server

This directory contains comprehensive end-to-end tests for the Launchpad MCP API server, testing the complete workflow from template creation to contract deployment.

## Test Structure

### Files
- `testutils.go` - Test utilities and setup helpers
- `test_contracts.go` - Sample Solidity contracts for testing
- `api_server_test.go` - Main API server integration tests
- `ethereum_integration_test.go` - Ethereum blockchain integration tests
- `address.go` - Test account private keys
- `contracts/` - Sample Solidity contract files
- `api/` - Playwright browser-based E2E tests
  - `uniswap_test.go` - Uniswap deployment browser tests
  - `page_objects.go` - Page object models for UI interactions
  - `test_helpers.go` - Playwright-specific test utilities
  - `wallet_provider.js` - EIP-6963 compliant test wallet provider

### Test Categories

#### 1. API Server Tests (`api_server_test.go`)
- **Template Workflow**: Creating templates and testing the launch deployment process
- **Error Handling**: Testing invalid sessions, expired sessions, and malformed data
- **Session Management**: Testing session lifecycle and concurrent operations
- **Database Integration**: Testing all CRUD operations with the database

#### 2. Ethereum Integration Tests (`ethereum_integration_test.go`)
- **Contract Deployment**: Testing actual contract deployment workflow
- **Full Workflow**: End-to-end testing from template creation to deployment confirmation
- **Blockchain Interaction**: Testing with real Ethereum testnet connections

#### 3. Playwright Browser Tests (`api/uniswap_test.go`)
- **Browser-based UI Testing**: Complete user workflow testing with real browser
- **EIP-6963 Wallet Integration**: Test wallet discovery and connection
- **Real Transaction Signing**: Test wallet transaction signing with real blockchain
- **Visual Verification**: Test UI state changes and user interactions
- **Error Handling**: Test browser-specific error scenarios

## Running the Tests

### Prerequisites

1. **Go Dependencies**: All dependencies should be installed via `go mod tidy`
2. **Local Ethereum Testnet**: For integration tests, anvil should be running on `localhost:8545`
3. **Playwright Browsers**: For browser tests, install Playwright browsers with `make playwright-install`

### Basic Tests (No Blockchain Required)

```bash
# Run all API server tests (no Ethereum required)
go test ./e2e -v -run "TestAPIServer"

# Run specific test categories
go test ./e2e -v -run "TestAPIServer_TemplateWorkflow"
go test ./e2e -v -run "TestAPIServer_ErrorHandling"
go test ./e2e -v -run "TestAPIServer_SessionManagement"
```

### Integration Tests (Ethereum Required)

```bash
# Start local Ethereum testnet (in separate terminal)
make e2e-network
# or manually: anvil

# Run Ethereum integration tests
go test ./e2e -v -run "TestEthereumIntegration"

# Run all tests including integration
go test ./e2e -v
```

### Using Make Commands

```bash
# Start anvil testnet
make e2e-network

# Run tests (in another terminal)
make test

# Run Playwright browser tests
make e2e-playwright

# Run all E2E tests including Playwright
make e2e-all
```

### Playwright Browser Tests (Ethereum Required)

```bash
# Install Playwright browsers (one-time setup)
make playwright-install

# Start local Ethereum testnet (in separate terminal)
make e2e-network

# Run Playwright E2E tests
make e2e-playwright

# Run specific Playwright tests
go test ./e2e/api -v -run "TestUniswapDeploymentPlaywright"
go test ./e2e/api -v -run "TestUniswapDeploymentErrorHandling"
```

## Test Configuration

### Test Accounts
The tests use predefined test accounts with known private keys:
- `TESTING_PK_1`: Primary test account for deployment testing
- `TESTING_PK_2`: Secondary test account for concurrent testing

These accounts are automatically funded when using anvil with default settings.

### Test Database
Each test creates an isolated SQLite database in a temporary directory, ensuring test isolation and cleanup.

### Test Server
The API server starts on a random available port for each test, preventing port conflicts during concurrent test execution.

## Test Scenarios

### 1. Template Creation and Management
- Create templates with valid Solidity code
- Validate template syntax and security
- Test template retrieval and listing
- Test template updates and deletion

### 2. Deployment Workflow
- Create deployment sessions via MCP tools
- Generate signing URLs for frontend interaction
- Test transaction session management
- Simulate user wallet interaction and signing
- Test deployment confirmation and database updates

### 3. API Endpoint Testing
- Test all HTTP endpoints (`/deploy/*`, `/api/deploy/*`, etc.)
- Validate request/response formats
- Test error conditions and edge cases
- Test concurrent session handling

### 4. Error Handling
- Invalid session IDs return 404
- Expired sessions are properly handled
- Wrong session types return appropriate errors
- Malformed request data is rejected

### 5. Blockchain Integration
- Connect to local Ethereum testnet
- Test account balance checking
- Simulate contract deployment transactions
- Test transaction confirmation workflow

### 6. Browser-based UI Testing (Playwright)
- Complete Uniswap deployment workflow in browser
- EIP-6963 wallet discovery and connection
- Real transaction signing and blockchain interaction
- Visual UI state verification
- Error handling and edge cases
- Screenshot capture on test failures

## Test Utilities

### TestSetup
The main test setup struct provides:
- Database initialization and cleanup
- API server startup and shutdown
- Ethereum client connection
- Test account management
- HTTP request helpers

### Key Helper Functions
- `NewTestSetup(t)` - Create complete test environment
- `CreateTestTemplate()` - Create test contract templates
- `MakeAPIRequest()` - Make HTTP requests to test server
- `VerifyEthereumConnection()` - Check testnet connectivity
- `GetPrimaryTestAccount()` - Get funded test account

### Playwright Test Utilities (api/)
- `NewPlaywrightTestSetup(t)` - Create browser test environment
- `NewUniswapDeploymentPage(page)` - Create page object for Uniswap deployment
- `InitializeTestWallet()` - Set up EIP-6963 wallet provider
- `CreateUniswapDeploymentSession()` - Create test session
- `TakeScreenshotOnFailure()` - Capture screenshots on test failure

## Mock vs Real Testing

### Mock Testing (Default)
- Uses simulated contract addresses and transaction hashes
- Tests API logic and database operations
- Fast execution, no external dependencies
- Suitable for CI/CD pipelines

### Real Testing (With Anvil)
- Connects to actual Ethereum testnet
- Tests real blockchain interactions
- Requires running anvil instance
- More comprehensive but slower

### Browser Testing (With Playwright)
- Real browser automation with Chromium
- Complete user workflow testing
- EIP-6963 wallet provider simulation
- Real transaction signing and blockchain submission
- Visual verification of UI state changes
- Most comprehensive but slowest execution

## Environment Variables

No special environment variables are required. The tests use hardcoded values suitable for local development:
- RPC URL: `http://localhost:8545`
- Chain ID: `31337` (Anvil default)
- Test accounts: Anvil's default funded accounts

## Debugging

### Verbose Output
```bash
go test ./e2e -v -run "TestName"
```

### Database Inspection
Test databases are created in temporary directories. To inspect:
```bash
# Find temp database location from test output
sqlite3 /tmp/launchpad-test-*/test.db
```

### API Server Logs
The API server runs with logging enabled during tests. HTTP requests and responses are logged to help with debugging.

### Playwright Debugging
For browser tests:
```bash
# Run tests with visible browser (set headless: false in test_helpers.go)
go test ./e2e/api -v -run "TestUniswapDeploymentPlaywright"

# Screenshots are automatically captured on test failures
# Look for files like: screenshot_uniswap_deployment_*.png
```

## Best Practices

1. **Test Isolation**: Each test creates its own database and server instance
2. **Cleanup**: Always use `defer setup.Cleanup()` in tests
3. **Error Checking**: All database and API operations include error checking
4. **Timeouts**: Network operations use appropriate timeouts
5. **Skip Logic**: Integration tests skip gracefully when anvil is not available

## Extending Tests

To add new test scenarios:

1. **New API Endpoints**: Add tests to `api_server_test.go`
2. **New Contract Types**: Add contracts to `test_contracts.go` and `contracts/`
3. **New Blockchain Features**: Add tests to `ethereum_integration_test.go`
4. **New Browser Workflows**: Add tests to `api/uniswap_test.go` or create new test files
5. **New Utilities**: Add helper functions to `testutils.go` or `api/test_helpers.go`

Example API Test:
```go
func TestNewFeature(t *testing.T) {
    setup := NewTestSetup(t)
    defer setup.Cleanup()
    
    // Your test logic here
}
```

Example Playwright Test:
```go
func TestNewBrowserFeature(t *testing.T) {
    setup := NewPlaywrightTestSetup(t)
    defer setup.Cleanup()
    defer setup.TakeScreenshotOnFailure(t, "new_feature")
    
    // Browser test logic here
}
```

## Playwright Test Architecture

### EIP-6963 Wallet Provider
The Playwright tests use a custom JavaScript wallet provider (`wallet_provider.js`) that:
- Implements the EIP-6963 wallet discovery standard
- Provides a complete Ethereum provider interface
- Communicates with Go test code for transaction signing
- Uses real private keys and blockchain interactions

### Page Object Model
Tests use page object models (`page_objects.go`) for:
- Reusable UI interaction methods
- Consistent element selectors
- Better test maintainability
- Separation of test logic from UI details

### Browser Configuration
- Headless mode by default (configurable)
- 1280x720 viewport for consistent rendering
- Chromium browser engine
- Automatic screenshot capture on failures