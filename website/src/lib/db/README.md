# Database Setup

This directory contains the database schema, connection utilities, and migration scripts for the authentication system.

## Files Overview

- `schema.ts` - Drizzle ORM schema definitions for all database tables
- `connection.ts` - Database connection setup with error handling and pooling
- `queries.ts` - Type-safe query functions for common database operations
- `migrate.ts` - Migration utilities for database setup and updates
- `index.ts` - Main export file for all database utilities

## Database Schema

The authentication system uses the following tables:

### Core Tables

- **users** - User profiles from OAuth providers
- **sessions** - Auth.js session management
- **accounts** - OAuth provider account links
- **verification_tokens** - Email verification and password reset

### Authentication Tables

- **authenticators** - WebAuthn passkey credentials
- **jwt_tokens** - Custom JWT tokens created by users

## Setup Instructions

### 1. Environment Configuration

Copy `.env.example` to `.env.local` and configure your database:

```bash
DATABASE_URL=postgresql://user:password@localhost:5432/dbname
```

### 2. Generate Migration

Generate the initial migration (already done):

```bash
pnpm db:generate
```

### 3. Run Migration

Apply migrations to your database:

```bash
pnpm db:migrate
```

Or use the initialization script:

```bash
pnpm db:init
```

### 4. Database Studio (Optional)

Open Drizzle Studio to view your database:

```bash
pnpm db:studio
```

## Usage Examples

### Basic Queries

```typescript
import { createUser, getUserById, createSession } from "@/lib/db/queries";

// Create a new user
const user = await createUser({
  id: "user-123",
  email: "user@example.com",
  name: "John Doe",
});

// Get user by ID
const foundUser = await getUserById("user-123");

// Create a session
const session = await createSession({
  sessionToken: "session-token",
  userId: "user-123",
  expires: new Date(Date.now() + 24 * 60 * 60 * 1000), // 24 hours
});
```

### JWT Token Management

```typescript
import {
  createJwtToken,
  getUserJwtTokens,
  deactivateJwtToken,
} from "@/lib/db/queries";

// Create a JWT token
const token = await createJwtToken({
  id: "token-123",
  userId: "user-123",
  tokenName: "API Access Token",
  jti: "unique-jti",
  aud: ["api.example.com"],
  roles: ["user"],
  scopes: ["read", "write"],
});

// Get user's tokens
const userTokens = await getUserJwtTokens("user-123");

// Deactivate a token
await deactivateJwtToken("token-123");
```

## Database Connection Health

```typescript
import { checkDatabaseConnection } from "@/lib/db/connection";

const isHealthy = await checkDatabaseConnection();
console.log("Database connected:", isHealthy);
```

## Migration Management

```typescript
import { runMigrations, initializeDatabase } from "@/lib/db/migrate";

// Run migrations programmatically
await runMigrations();

// Initialize fresh database
await initializeDatabase();
```

## Error Handling

All query functions include proper error handling and will throw descriptive errors:

- Connection errors are logged and re-thrown
- Database constraint violations are caught and handled
- All functions include try-catch blocks with meaningful error messages

## Type Safety

All database operations are fully type-safe using Drizzle ORM:

- Schema types are automatically inferred
- Query results are properly typed
- Insert/update operations validate data structure
- Foreign key relationships are enforced at the type level
