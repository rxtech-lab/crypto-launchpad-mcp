# Requirements Document

## Introduction

This feature implements a comprehensive authentication system using Auth.js with support for passkey and Google sign-in methods. The system includes a user dashboard for JWT token management and session control, integrated into the existing Next.js website with shadcn/ui styling and Drizzle ORM for database operations.

During the UI implementation, use playwright mcp for UI debugging.

## Requirements

### Requirement 1

**User Story:** As a visitor, I want to sign up or sign in using passkeys or Google authentication, so that I can securely access my dashboard without remembering passwords.

#### Acceptance Criteria

1. WHEN a user visits the authentication page THEN the system SHALL display options for passkey and Google authentication
2. WHEN a new user selects passkey authentication THEN the system SHALL initiate WebAuthn registration flow using Auth.js WebAuthn provider for account creation
3. WHEN an existing user selects passkey authentication THEN the system SHALL initiate WebAuthn authentication flow using Auth.js WebAuthn provider for login
4. WHEN a user selects Google authentication THEN the system SHALL redirect to Google OAuth flow and create account if new user
5. WHEN authentication or registration is successful THEN the system SHALL create a session and redirect to the dashboard
6. WHEN authentication fails THEN the system SHALL display appropriate error messages
7. IF a user is already authenticated THEN the system SHALL redirect them to the dashboard
8. WHEN a new user completes registration THEN the system SHALL automatically sign them in

### Requirement 2

**User Story:** As an authenticated user, I want to access a dashboard where I can manage my JWT tokens and sessions, so that I can control my authentication state and generate tokens for API access.

#### Acceptance Criteria

1. WHEN an authenticated user accesses the dashboard THEN the system SHALL display their user information
2. WHEN a user requests to create a JWT token THEN the system SHALL generate a token with the specified AuthenticatedUser structure
3. WHEN a user creates a JWT token THEN the system SHALL store the session information in the database
4. WHEN a user views their dashboard THEN the system SHALL display all active sessions
5. WHEN a user deletes a session THEN the system SHALL remove it from the database and invalidate the token
6. IF an unauthenticated user tries to access the dashboard THEN the system SHALL redirect them to the sign-in page

### Requirement 3

**User Story:** As a user of the existing website, I want to see a dashboard button in the navigation, so that I can easily access my authentication dashboard or sign up if I'm new.

#### Acceptance Criteria

1. WHEN a user visits the website THEN the system SHALL display a dashboard button in the navigation
2. WHEN an unauthenticated user clicks the dashboard button THEN the system SHALL redirect them to the authentication page with sign-up/sign-in options
3. WHEN an authenticated user clicks the dashboard button THEN the system SHALL redirect them to the dashboard
4. WHEN a user is on the dashboard THEN the system SHALL provide a way to return to the main website
5. WHEN a new user first clicks the dashboard button THEN the system SHALL guide them through the sign-up flow

### Requirement 4

**User Story:** As a developer, I want the authentication system to use proper database schema with Drizzle ORM, so that user sessions and tokens are properly persisted and managed.

#### Acceptance Criteria

1. WHEN the system initializes THEN it SHALL create proper database tables for users, sessions, and tokens
2. WHEN a user authenticates THEN the system SHALL store user information according to the AuthenticatedUser structure
3. WHEN a JWT token is created THEN the system SHALL store session metadata in the database
4. WHEN a session is deleted THEN the system SHALL properly clean up database records
5. WHEN the system queries user data THEN it SHALL use Drizzle ORM for type-safe database operations

### Requirement 5

**User Story:** As a developer, I want the code to be well-organized with proper separation of concerns, so that the authentication logic is maintainable and testable.

#### Acceptance Criteria

1. WHEN implementing authentication logic THEN the system SHALL use custom hooks for state management
2. WHEN creating UI components THEN the system SHALL use functional components with proper prop typing
3. WHEN handling authentication flows THEN the system SHALL separate business logic into utility functions
4. WHEN styling components THEN the system SHALL use shadcn/ui components consistently
5. WHEN implementing database operations THEN the system SHALL abstract queries into service functions
