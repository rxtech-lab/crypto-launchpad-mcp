# Implementation Plan

- [x] 1. Set up project dependencies and configuration

  - Install Auth.js, Drizzle ORM, TanStack Query, and testing dependencies
  - Configure environment variables and database connection
  - Set up Vitest configuration for component testing
  - _Requirements: 4.1, 4.5_

- [x] 2. Create database schema and connection setup

  - Define Drizzle schema for users, sessions, JWT tokens, and authenticators tables
  - Set up database connection utilities with proper error handling
  - Create database migration scripts
  - _Requirements: 4.1, 4.2, 4.4_

- [x] 3. Configure Auth.js with WebAuthn and Google providers

  - Set up Auth.js configuration with WebAuthn and Google OAuth providers
  - Configure session strategy and authentication callbacks
  - Create Auth.js API routes for authentication handling
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 4. Implement authentication middleware and route protection

  - Create Next.js middleware for route protection
  - Set up session validation and redirect logic
  - Test middleware functionality with protected and public routes
  - _Requirements: 2.6, 3.2, 3.3_

- [x] 5. Create authentication pages and components
- [x] 5.1 Build authentication page layout with redirect logic

  - Create auth layout that redirects authenticated users to dashboard
  - Implement authentication page with sign-in/sign-up options
  - Add error handling page for authentication failures
  - _Requirements: 1.1, 1.5, 3.2_

- [x] 5.2 Implement WebAuthn authentication component

  - Create WebAuthn button component for passkey authentication
  - Handle WebAuthn registration flow for new users
  - Handle WebAuthn authentication flow for existing users
  - _Requirements: 1.2, 1.3, 1.8_

- [x] 5.3 Implement Google OAuth authentication component

  - Create Google sign-in button component
  - Handle OAuth flow and account creation for new users
  - Integrate with Auth.js Google provider configuration
  - _Requirements: 1.4, 1.8_

- [x] 6. Create protected dashboard layout and components
- [x] 6.1 Build protected dashboard layout

  - Create dashboard layout that redirects unauthenticated users
  - Implement navigation between dashboard and main website
  - Add user profile display component
  - _Requirements: 2.1, 2.6, 3.3, 3.4_

- [x] 6.2 Implement JWT token management functionality

  - Create JWT token generation utilities with AuthenticatedUser structure
  - Build token creation form component with proper validation
  - Implement token display and management interface
  - _Requirements: 2.2, 2.3, 4.2, 4.3_

- [x] 6.3 Create session management components

  - Build session list component to display active sessions
  - Implement session deletion functionality with database cleanup
  - Add session metadata display and management
  - _Requirements: 2.4, 2.5, 4.4_

- [x] 7. Implement server actions for token and session operations

  - Create server actions for JWT token CRUD operations in tokens folder
  - Implement server actions for session management operations
  - Add proper error handling and validation for server actions
  - _Requirements: 2.2, 2.3, 2.5, 4.4_

- [x] 8. Create custom hooks with TanStack Query integration
- [x] 8.1 Build authentication state management hook

  - Create useAuth hook for authentication state management
  - Integrate with Auth.js session handling
  - Add authentication status and user information access
  - _Requirements: 5.1, 5.2_

- [x] 8.2 Implement token management hooks with TanStack Query

  - Create useTokens hook with TanStack Query for token operations
  - Implement optimistic updates and cache management
  - Add error handling and loading states for token operations
  - _Requirements: 5.1, 5.3_

- [x] 8.3 Create session management hooks with TanStack Query

  - Build useSessions hook with TanStack Query for session operations
  - Implement real-time session updates and cache invalidation
  - Add proper error handling for session management
  - _Requirements: 5.1, 5.3_

- [x] 9. Update existing website navigation

  - Add dashboard button component to existing website navigation
  - Implement conditional rendering based on authentication status
  - Style dashboard button using existing shadcn/ui components
  - _Requirements: 3.1, 3.2, 3.3, 5.4_

- [x] 10. Create TypeScript type definitions

  - Define AuthenticatedUser interface matching the specified structure
  - Create database types for Drizzle schema
  - Add authentication and JWT token type definitions
  - _Requirements: 4.2, 5.2_

- [-] 11. Write comprehensive tests
- [x] 11.1 Create unit tests for utility functions

  - Test JWT token generation and validation functions
  - Test database query functions and error handling
  - Test authentication helper functions
  - _Requirements: 5.5_

- [x] 11.2 Write component tests using Vitest

  - Test authentication form interactions and validation
  - Test dashboard component rendering and user interactions
  - Test navigation button behavior and conditional rendering
  - _Requirements: 5.4, 5.5_

- [x] 12. Set up TanStack Query provider and configuration
  - Configure QueryClient with proper defaults and error handling
  - Set up React Query DevTools for development
  - Integrate query provider with Next.js app structure
  - _Requirements: 5.1, 5.3_
