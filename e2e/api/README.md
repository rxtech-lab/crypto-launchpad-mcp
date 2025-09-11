# Authentication E2E Tests

This directory contains comprehensive end-to-end tests for JWT and OAuth authentication in the API server.

## Test Files

- `auth_test.go` - Main authentication test suite with comprehensive scenarios
- `auth_helpers.go` - Authentication helper functions and utilities for testing
- `README.md` - This documentation file

## Test Coverage

### JWT Authentication Tests
- ✅ Valid JWT token authentication
- ✅ Expired JWT token rejection
- ✅ Malformed JWT token rejection
- ✅ Missing authorization header handling
- ✅ JWT token with custom claims (roles, scopes)
- ✅ JWT secret configuration validation
- ✅ Authentication middleware order testing

### OAuth Authentication Tests
- ✅ Mock OAuth server setup and validation
- ✅ OAuth token validation scenarios
- ✅ OAuth environment configuration testing
- ✅ JWKS endpoint mocking

### Authentication Bypass Tests
- ✅ Health endpoint bypasses authentication
- ✅ Transaction page routes (`/tx/*`) bypass authentication
- ✅ Well-known OAuth endpoints bypass authentication
- ✅ Static assets bypass authentication

### Server Integration Tests
- ✅ WWW-Authenticate headers on unauthorized requests
- ✅ Environment variable configuration for OAuth
- ✅ Middleware order verification (OAuth then JWT)

## Prerequisites

Before running these tests, ensure you have:

1. **Ethereum Testnet Running**:
   ```bash
   make e2e-network
   ```
   This starts an Anvil testnet on `localhost:8545` required for blockchain operations.

2. **Go Dependencies**:
   ```bash
   go mod tidy
   ```

## Running the Tests

### Run All Authentication Tests
```bash
go test -v ./e2e/api -timeout 30s
```

### Run Specific Test Categories
```bash
# JWT Authentication Tests
go test -v ./e2e/api -run TestJWTAuth -timeout 30s

# OAuth Authentication Tests  
go test -v ./e2e/api -run TestOAuthAuth -timeout 30s

# Authentication Bypass Tests
go test -v ./e2e/api -run TestAuthBypass -timeout 30s
```

### Run Individual Tests
```bash
# Test valid JWT token authentication
go test -v ./e2e/api -run TestJWTAuth_TransactionAPI_ValidToken -timeout 30s

# Test OAuth configuration
go test -v ./e2e/api -run TestOAuthConfiguration_EnvironmentVariables -timeout 30s
```

## Test Architecture

### TestSetup Structure
The tests use a `TestSetup` structure that provides:
- In-memory SQLite database for testing
- Real Ethereum client connected to testnet
- All necessary service layers (DB, Chain, Template, Deployment, Transaction)
- Random port allocation for test servers

### AuthTestHelper
The `AuthTestHelper` provides utilities for:
- JWT token creation with various scenarios
- Mock OAuth/JWKS server setup
- Environment variable management
- Common authentication test scenarios

### Server Configuration
Tests create a real API server with:
- Authentication middleware enabled
- Mock OAuth/JWKS endpoints
- Real transaction sessions for testing
- Proper environment variable configuration

## Environment Variables

The tests configure these environment variables using `t.Setenv()` for proper test isolation:

```bash
JWT_SECRET="test-jwt-secret-for-e2e-testing"
SCALEKIT_ENV_URL="http://localhost:<port>/.well-known/jwks.json"
SCALEKIT_RESOURCE_METADATA_URL="http://localhost:<port>/metadata"  
OAUTH_AUTHENTICATION_SERVER="http://localhost:<port>"
OAUTH_RESOURCE_URL="http://localhost:3000/api"
OAUTH_RESOURCE_DOCUMENTATION_URL="http://localhost:3000/docs"
```

All environment variables are automatically cleaned up after each test using Go's `t.Setenv()` functionality.

## Test Scenarios

### JWT Token Scenarios
1. **Standard User Token**: Basic user with read permissions
2. **Admin User Token**: Admin user with full permissions
3. **Service Account Token**: Long-lived service account token
4. **Expired Token**: Token that has passed its expiration time
5. **Malformed Token**: Invalid JWT format
6. **Missing Claims Token**: Token missing required claims

### API Endpoints Tested
- `POST /api/tx/:session_id/transaction/:index` - Requires authentication
- `GET /health` - Bypasses authentication
- `GET /tx/:session_id` - Bypasses authentication
- `GET /.well-known/oauth-protected-resource/mcp` - Bypasses authentication
- `GET /static/tx/app.js` - Bypasses authentication

### Mock OAuth Server
The tests include a complete mock OAuth server that provides:
- JWKS endpoint (`/.well-known/jwks.json`)
- Resource metadata endpoint (`/metadata`)
- Token validation endpoint (`/validate`)

## Integration with API Server

These tests verify the complete authentication flow:

1. **Request arrives** at API server
2. **OAuth middleware** attempts token validation first
3. **JWT middleware** provides fallback authentication
4. **Authenticated user** stored in Fiber context
5. **Request processed** with proper authorization context

## Error Scenarios

The tests verify proper error handling for:
- Missing authorization headers
- Invalid token formats
- Expired tokens
- Malformed tokens
- Missing JWT secrets
- OAuth configuration errors
- Network timeouts

## Debugging

If tests fail, check:

1. **Testnet Running**: Ensure `make e2e-network` is running
2. **Port Conflicts**: Tests use random ports but conflicts can occur
3. **Environment**: Clean environment variables between test runs
4. **Logs**: Test output includes detailed error messages and server logs

## Contributing

When adding new authentication tests:

1. Use the `AuthTestHelper` for common operations
2. Follow the existing naming convention: `TestJWTAuth_*` or `TestOAuthAuth_*`
3. Include both success and failure scenarios
4. Test with real API endpoints, not mocked handlers
5. Verify proper cleanup in teardown methods
6. Add documentation for new test scenarios