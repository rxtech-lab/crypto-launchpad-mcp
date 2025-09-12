import type {
  User,
  NewUser,
  Session,
  NewSession,
  Account,
  NewAccount,
  JwtToken,
  NewJwtToken,
  Authenticator,
  NewAuthenticator,
  VerificationToken,
  NewVerificationToken,
} from "@/lib/db/schema";

// Re-export all database types for easier imports
export type {
  User,
  NewUser,
  Session,
  NewSession,
  Account,
  NewAccount,
  JwtToken,
  NewJwtToken,
  Authenticator,
  NewAuthenticator,
  VerificationToken,
  NewVerificationToken,
};

// Database operation result types
export interface DatabaseResult<T> {
  success: boolean;
  data?: T;
  error?: string;
}

// Query options for pagination and filtering
export interface QueryOptions {
  limit?: number;
  offset?: number;
  orderBy?: string;
  orderDirection?: "asc" | "desc";
}

// User session with additional metadata
export interface UserSessionWithMetadata extends Session {
  user: User;
  isCurrentSession?: boolean;
}

// JWT token with user information
export interface JwtTokenWithUser extends JwtToken {
  user: Pick<User, "id" | "name" | "email">;
}

// Database connection status
export interface DatabaseStatus {
  connected: boolean;
  lastChecked: Date;
  error?: string;
}

// Server action result types
export interface ActionResult<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

// Token action specific result types
export interface TokenActionResult extends ActionResult {
  data?: {
    tokenRecord?: JwtToken;
    jwt?: string;
    authenticatedUser?: import("./auth").AuthenticatedUser;
    tokens?: JwtToken[];
    message?: string;
  };
}

// Session action specific result types
export interface SessionActionResult extends ActionResult {
  data?: {
    sessions?: UserSessionWithMetadata[];
    session?: UserSessionWithMetadata;
    message?: string;
    deletedCount?: number;
    isValid?: boolean;
    reason?: string;
    isCurrent?: boolean;
  };
}

// API response types for client-side hooks
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// Pagination types for future use
export interface PaginationOptions {
  page?: number;
  limit?: number;
  offset?: number;
}

export interface PaginatedResult<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
    hasNext: boolean;
    hasPrev: boolean;
  };
}

// Filter types for queries
export interface TokenFilters {
  isActive?: boolean;
  clientId?: string;
  roles?: string[];
  scopes?: string[];
  createdAfter?: Date;
  createdBefore?: Date;
  expiresAfter?: Date;
  expiresBefore?: Date;
}

export interface SessionFilters {
  isExpired?: boolean;
  createdAfter?: Date;
  createdBefore?: Date;
  expiresAfter?: Date;
  expiresBefore?: Date;
}
