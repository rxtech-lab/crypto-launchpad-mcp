# Server Actions Implementation

This document provides an overview of all server actions implemented for JWT token and session management operations.

## Token Actions (`src/app/dashboard/tokens/actions.ts`)

### `createToken(data: unknown): Promise<TokenActionResult>`

- **Purpose**: Create a new JWT token for the authenticated user
- **Validation**: Uses Zod schema validation for input data
- **Security**: Verifies user authentication, validates token parameters
- **Returns**: Token record, JWT string, and authenticated user object
- **Requirements**: 2.2, 2.3

### `getTokens(): Promise<TokenActionResult>`

- **Purpose**: Retrieve all active JWT tokens for the authenticated user
- **Security**: Ensures user can only access their own tokens
- **Returns**: Array of user's JWT tokens
- **Requirements**: 2.2

### `deactivateToken(tokenId: unknown): Promise<TokenActionResult>`

- **Purpose**: Soft delete a JWT token (marks as inactive)
- **Validation**: Validates token ID format (UUID)
- **Security**: Verifies token ownership before deactivation
- **Side Effects**: Revalidates tokens page cache
- **Requirements**: 2.2, 2.5

### `removeToken(tokenId: unknown): Promise<TokenActionResult>`

- **Purpose**: Permanently delete a JWT token from database
- **Validation**: Validates token ID format (UUID)
- **Security**: Verifies token ownership before deletion
- **Side Effects**: Revalidates tokens page cache
- **Requirements**: 2.2, 2.5

### `getTokenByJti(jti: unknown): Promise<TokenActionResult>`

- **Purpose**: Retrieve a specific JWT token by its JTI for validation
- **Validation**: Validates JTI format (UUID)
- **Security**: Verifies token ownership
- **Use Case**: Token validation and lookup operations
- **Requirements**: 2.3

## Session Actions (`src/app/dashboard/sessions/actions.ts`)

### `getSessions(): Promise<SessionActionResult>`

- **Purpose**: Retrieve all sessions for the authenticated user with metadata
- **Security**: User can only access their own sessions
- **Returns**: Sessions with metadata (user agent, IP, current session flag)
- **Requirements**: 2.5

### `removeSession(sessionToken: unknown): Promise<SessionActionResult>`

- **Purpose**: Delete a specific session by session token
- **Validation**: Validates session token format and ownership
- **Security**: Prevents deletion of current session, verifies ownership
- **Side Effects**: Revalidates sessions page cache
- **Requirements**: 2.5

### `removeOtherSessions(): Promise<SessionActionResult>`

- **Purpose**: Delete all sessions except the current one
- **Security**: Preserves current session, only affects user's own sessions
- **Use Case**: Security feature to log out from all other devices
- **Side Effects**: Revalidates sessions page cache
- **Requirements**: 2.5

### `getSessionDetails(sessionToken: unknown): Promise<SessionActionResult>`

- **Purpose**: Get detailed information about a specific session
- **Validation**: Validates session token format
- **Security**: Verifies session ownership
- **Returns**: Session with metadata and current session flag
- **Requirements**: 2.5

### `validateSession(sessionToken: unknown): Promise<SessionActionResult>`

- **Purpose**: Validate if a session token is valid and active
- **Validation**: Checks token format, existence, and expiration
- **Security**: Verifies session ownership
- **Returns**: Validation result with session details if valid
- **Requirements**: 2.5

## Auth Actions (`src/lib/actions/auth-actions.ts`)

### `getCurrentUser(): Promise<AuthActionResult>`

- **Purpose**: Get the current authenticated user's profile information
- **Security**: Requires valid authentication
- **Returns**: User profile data
- **Requirements**: 4.4

### `signOutUser(): Promise<void>`

- **Purpose**: Sign out the current user and redirect to home page
- **Security**: Clears authentication session
- **Side Effects**: Redirects to home page
- **Requirements**: 4.4

### `validateAuth(): Promise<AuthActionResult>`

- **Purpose**: Validate current authentication status
- **Returns**: Authentication status and user information
- **Use Case**: Check if user is still authenticated
- **Requirements**: 4.4

### `refreshUserData(path?: string): Promise<AuthActionResult>`

- **Purpose**: Refresh cached user data after operations
- **Side Effects**: Revalidates specified paths or default dashboard paths
- **Use Case**: Update UI after data changes
- **Requirements**: 4.4

## Validation Utilities

### Token Validation (`src/lib/validation/token-validation.ts`)

- **createTokenSchema**: Zod schema for token creation input
- **tokenIdSchema**: UUID validation for token IDs
- **jtiSchema**: UUID validation for JTI values
- **Validation Functions**: `validateCreateToken`, `validateTokenId`, `validateJti`

### Session Validation (`src/lib/validation/session-validation.ts`)

- **sessionTokenSchema**: Session token format validation
- **validateSessionToken**: Session token validation function
- **validateSessionOperation**: Prevents invalid operations (e.g., deleting current session)

## Error Handling

### Comprehensive Error Handling (`src/lib/utils/action-errors.ts`)

- **ActionErrorHandler**: Centralized error handling class
- **Database Errors**: Handles constraint violations, foreign key errors
- **Authentication Errors**: Handles access denied, session required
- **Validation Errors**: Handles Zod validation failures
- **JWT Errors**: Handles token expiration, invalid format
- **Generic Errors**: Fallback for unexpected errors

### Error Response Format

```typescript
interface ActionResult {
  success: boolean;
  error?: string;
  errorCode?: string;
  errorDetails?: any;
  data?: any;
  message?: string;
}
```

## Security Features

1. **Authentication Verification**: All actions verify user authentication
2. **Authorization Checks**: Users can only access/modify their own data
3. **Input Validation**: Comprehensive validation using Zod schemas
4. **Session Protection**: Prevents deletion of current session
5. **Error Sanitization**: Errors are sanitized before returning to client
6. **Cache Invalidation**: Proper cache revalidation after mutations

## Usage Examples

```typescript
// Create a new token
const result = await createToken({
  tokenName: "API Access Token",
  expiresIn: "30d",
  scopes: ["read", "write"],
  roles: ["user"],
});

// Remove a session
const sessionResult = await removeSession("session-token-here");

// Get all user sessions
const sessions = await getSessions();
```

## Requirements Coverage

- **Requirement 2.2**: JWT token CRUD operations ✅
- **Requirement 2.3**: JWT token generation and validation ✅
- **Requirement 2.5**: Session management operations ✅
- **Requirement 4.4**: Proper database operations with error handling ✅
