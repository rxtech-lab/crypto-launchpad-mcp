# Design Document

## Overview

This design implements a comprehensive authentication system using Auth.js v5 with WebAuthn (passkey) and Google OAuth providers. The system integrates seamlessly into the existing Next.js website, adding authentication capabilities while maintaining the current design aesthetic using shadcn/ui components.

The architecture follows Next.js App Router patterns with server-side authentication, client-side state management through custom hooks with TanStack Query for data fetching, and persistent storage using Drizzle ORM with a PostgreSQL database.

## Architecture

### High-Level Architecture

```mermaid
graph TB
    A[User] --> B[Next.js App Router]
    B --> C[Auth.js Middleware]
    C --> D[Authentication Pages]
    C --> E[Protected Dashboard]

    D --> F[WebAuthn Provider]
    D --> G[Google OAuth Provider]

    E --> H[JWT Token Management]
    E --> I[Session Management]

    H --> J[Drizzle ORM]
    I --> J
    J --> K[PostgreSQL Database]

    B --> L[Existing Website Components]
    L --> M[Updated Navigation]
```

### Authentication Flow

```mermaid
sequenceDiagram
    participant U as User
    participant N as Navigation
    participant A as Auth Page
    participant P as Provider
    participant D as Dashboard
    participant DB as Database

    U->>N: Click Dashboard Button
    N->>A: Redirect to /auth
    A->>U: Show Sign In/Up Options
    U->>P: Choose Provider (WebAuthn/Google)
    P->>A: Authentication Result
    A->>DB: Store User Session
    A->>D: Redirect to Dashboard
    D->>DB: Query User Sessions
    D->>U: Display Dashboard
```

## Components and Interfaces

### Core Authentication Components

#### 1. Auth Configuration (`src/auth.ts`)

- Configure Auth.js with WebAuthn and Google providers
- Define session strategy and callbacks
- Handle user creation and session management

#### 2. Authentication Pages

- `/auth` - Unified sign-in/sign-up page
- `/auth/error` - Authentication error handling
- `/auth/verify-request` - Email verification (if needed)

#### 3. Dashboard Components

- `DashboardLayout` - Protected layout wrapper (redirects unauthenticated users)
- `UserProfile` - Display user information
- `TokenManager` - JWT token creation and management
- `SessionList` - Active sessions display and management

#### 4. Auth Layout

- `AuthLayout` - Authentication layout wrapper (redirects authenticated users)

#### 5. Navigation Updates

- `DashboardButton` - Navigation button component
- Updated layout to include dashboard access

### Database Schema (Drizzle ORM)

```typescript
// User table
export const users = pgTable("users", {
  id: text("id").primaryKey(),
  email: text("email").unique(),
  name: text("name"),
  image: text("image"),
  createdAt: timestamp("created_at").defaultNow(),
  updatedAt: timestamp("updated_at").defaultNow(),
});

// Sessions table (Auth.js sessions)
export const sessions = pgTable("sessions", {
  sessionToken: text("session_token").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  expires: timestamp("expires").notNull(),
});

// JWT Tokens table (custom tokens created by users)
export const jwtTokens = pgTable("jwt_tokens", {
  id: text("id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  tokenName: text("token_name").notNull(),
  jti: text("jti").unique().notNull(),
  aud: json("aud").$type<string[]>(),
  clientId: text("client_id"),
  roles: json("roles").$type<string[]>().default([]),
  scopes: json("scopes").$type<string[]>().default([]),
  createdAt: timestamp("created_at").defaultNow(),
  expiresAt: timestamp("expires_at"),
  isActive: boolean("is_active").default(true),
});

// WebAuthn Authenticators table
export const authenticators = pgTable("authenticators", {
  credentialID: text("credential_id").primaryKey(),
  userId: text("user_id")
    .notNull()
    .references(() => users.id, { onDelete: "cascade" }),
  providerAccountId: text("provider_account_id").notNull(),
  credentialPublicKey: text("credential_public_key").notNull(),
  counter: integer("counter").notNull(),
  credentialDeviceType: text("credential_device_type").notNull(),
  credentialBackedUp: boolean("credential_backed_up").notNull(),
  transports: text("transports"),
});
```

### AuthenticatedUser Type Implementation

```typescript
export interface AuthenticatedUser {
  aud: string[];
  client_id: string;
  exp: number;
  iat: number;
  iss: string;
  jti: string;
  nbf: number;
  oid: string;
  resid: string;
  roles: string[];
  scopes: string[];
  sid: string;
  sub: string;
}
```

## Data Models

### User Management

- **User Profile**: Basic user information from OAuth providers
- **Session Management**: Auth.js session handling with database persistence
- **JWT Token Metadata**: Custom token information and lifecycle management

### Token Structure

- **Standard JWT Claims**: `iss`, `sub`, `aud`, `exp`, `iat`, `nbf`, `jti`
- **Custom Claims**: `client_id`, `oid`, `resid`, `roles`, `scopes`, `sid`
- **Token Lifecycle**: Creation, validation, revocation, expiration

## Error Handling

### Authentication Errors

- **Provider Failures**: WebAuthn registration/authentication failures
- **OAuth Errors**: Google authentication issues
- **Session Errors**: Invalid or expired sessions

### Database Errors

- **Connection Issues**: Database connectivity problems
- **Constraint Violations**: Unique constraint failures
- **Transaction Failures**: Rollback scenarios

### JWT Token Errors

- **Generation Failures**: Token creation issues
- **Validation Errors**: Invalid token structure
- **Expiration Handling**: Expired token cleanup

## Testing Strategy

### Unit Tests

- **Authentication Hooks**: Custom React hooks for auth state
- **Utility Functions**: JWT generation and validation
- **Database Operations**: Drizzle ORM queries and mutations

### Component Tests (Vitest)

- **Authentication Pages**: Sign-in/sign-up form interactions
- **Dashboard Components**: Token management UI
- **Navigation Updates**: Dashboard button behavior

## Security Considerations

### Authentication Security

- **WebAuthn Implementation**: Secure passkey registration and authentication
- **OAuth Security**: Proper Google OAuth implementation with PKCE
- **Session Security**: Secure session token generation and storage

### JWT Token Security

- **Token Signing**: Secure JWT signing with proper algorithms
- **Scope Validation**: Proper role and scope enforcement
- **Token Revocation**: Immediate token invalidation capabilities

### Database Security

- **Connection Security**: Encrypted database connections
- **Data Validation**: Input sanitization and validation
- **Access Control**: Proper user data isolation

## Implementation Dependencies

### New Package Dependencies

```json
{
  "next-auth": "5.0.0-beta.25",
  "@auth/drizzle-adapter": "^1.7.4",
  "@simplewebauthn/server": "^12.0.0",
  "@simplewebauthn/browser": "^12.0.0",
  "drizzle-orm": "^0.36.4",
  "drizzle-kit": "^0.30.0",
  "postgres": "^3.4.5",
  "jsonwebtoken": "^9.0.2",
  "@types/jsonwebtoken": "^9.0.7",
  "uuid": "^11.0.3",
  "@types/uuid": "^10.0.0",
  "@tanstack/react-query": "^5.62.7",
  "@tanstack/react-query-devtools": "^5.62.7",
  "vitest": "^2.1.8",
  "@testing-library/react": "^16.1.0",
  "@testing-library/jest-dom": "^6.6.3",
  "jsdom": "^25.0.1"
}
```

### Environment Variables

```env
AUTH_SECRET=your-auth-secret
AUTH_URL=http://localhost:3000
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
DATABASE_URL=postgresql://user:password@localhost:5432/dbname
JWT_SECRET=your-jwt-secret
```

## File Structure

```
src/
├── auth.ts                          # Auth.js configuration
├── middleware.ts                    # Auth middleware
├── app/
│   ├── (protected)/
│   │   └── dashboard/
│   │       ├── page.tsx           # Dashboard main page
│   │       ├── layout.tsx         # Protected layout (redirects unauthenticated)
│   │       └── tokens/
│   │           ├── page.tsx       # Token management page
│   │           └── actions.ts    # Server actions for token operations
│   ├── auth/
│   │   ├── page.tsx               # Authentication page
│   │   ├── error/
│   │   │   └── page.tsx          # Auth error page
│   │   └── layout.tsx            # Auth layout (redirects authenticated)
│   └── api/
│       └── auth/
│           └── [...nextauth]/
│               └── route.ts       # Auth.js API routes
├── components/
│   ├── auth/
│   │   ├── sign-in-form.tsx       # Sign-in component
│   │   ├── webauthn-button.tsx    # WebAuthn authentication
│   │   └── google-sign-in.tsx     # Google OAuth button
│   ├── dashboard/
│   │   ├── dashboard-layout.tsx   # Dashboard wrapper
│   │   ├── user-profile.tsx       # User information display
│   │   ├── token-manager.tsx      # JWT token management
│   │   ├── session-list.tsx       # Active sessions
│   │   └── create-token-form.tsx  # Token creation form
│   └── navigation/
│       └── dashboard-button.tsx   # Navigation dashboard button
├── hooks/
│   ├── use-auth.ts                # Authentication state hook
│   ├── use-tokens.ts              # Token management hook (with TanStack Query)
│   └── use-sessions.ts            # Session management hook (with TanStack Query)
├── lib/
│   ├── auth/
│   │   ├── config.ts              # Auth configuration helpers
│   │   └── jwt.ts                 # JWT utilities
│   ├── db/
│   │   ├── schema.ts              # Drizzle schema
│   │   ├── connection.ts          # Database connection
│   │   └── queries.ts             # Database queries
│   ├── actions/
│   │   └── session-actions.ts     # Server actions for session operations
│   └── utils/
│       ├── token-generator.ts     # JWT token generation
│       └── session-manager.ts     # Session utilities
└── types/
    ├── auth.ts                    # Authentication types
    ├── jwt.ts                     # JWT token types
    └── database.ts                # Database types
```
