// API-specific type definitions

import type { JwtToken, Session, User } from "./database";
import type { AuthenticatedUser } from "./auth";

// Generic API response wrapper
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
  timestamp?: string;
}

// API error response
export interface ApiError {
  error: string;
  message?: string;
  code?: string;
  details?: Record<string, any>;
  timestamp: string;
}

// Tokens API types
export interface TokensApiResponse extends ApiResponse {
  data?: {
    tokens: JwtToken[];
  };
}

export interface CreateTokenApiRequest {
  tokenName: string;
  aud?: string[];
  clientId?: string;
  roles?: string[];
  scopes?: string[];
  expiresIn?: string;
}

export interface CreateTokenApiResponse extends ApiResponse {
  data?: {
    tokenRecord: JwtToken;
    jwt: string;
    authenticatedUser: AuthenticatedUser;
  };
}

export interface DeleteTokenApiResponse extends ApiResponse {
  data?: {
    message: string;
  };
}

// Sessions API types
export interface SessionsApiResponse extends ApiResponse {
  data?: {
    sessions: SessionWithMetadata[];
  };
}

export interface SessionWithMetadata extends Session {
  userAgent?: string;
  ipAddress?: string;
  deviceType?: string;
  browser?: string;
  os?: string;
  location?: string;
  lastActivity?: Date;
  isCurrent?: boolean;
}

export interface DeleteSessionApiResponse extends ApiResponse {
  data?: {
    message: string;
  };
}

export interface DeleteOtherSessionsApiRequest {
  currentSessionToken: string;
}

export interface DeleteOtherSessionsApiResponse extends ApiResponse {
  data?: {
    message: string;
    deletedCount: number;
  };
}

// Auth API types
export interface AuthApiResponse extends ApiResponse {
  data?: {
    user: User;
    session: Session;
  };
}

export interface SignInApiRequest {
  provider: "webauthn" | "google";
  credentials?: any;
  callbackUrl?: string;
}

export interface SignInApiResponse extends ApiResponse {
  data?: {
    url?: string;
    user?: User;
    session?: Session;
  };
}

export interface SignOutApiResponse extends ApiResponse {
  data?: {
    url?: string;
  };
}

// WebAuthn API types
export interface WebAuthnRegistrationRequest {
  username: string;
  displayName?: string;
}

export interface WebAuthnRegistrationResponse extends ApiResponse {
  data?: {
    options: PublicKeyCredentialCreationOptions;
  };
}

export interface WebAuthnVerificationRequest {
  credential: PublicKeyCredential;
  username?: string;
}

export interface WebAuthnVerificationResponse extends ApiResponse {
  data?: {
    verified: boolean;
    user?: User;
    session?: Session;
  };
}

// HTTP method types
export type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

// API endpoint configuration
export interface ApiEndpoint {
  method: HttpMethod;
  path: string;
  requiresAuth?: boolean;
  rateLimit?: {
    requests: number;
    window: number; // in seconds
  };
}

// Request context for API handlers
export interface ApiRequestContext {
  user?: User;
  session?: Session;
  ip?: string;
  userAgent?: string;
  method: HttpMethod;
  path: string;
  query: Record<string, string | string[]>;
  body?: any;
}

// API middleware types
export type ApiMiddleware = (
  context: ApiRequestContext
) => Promise<ApiRequestContext | ApiError>;

// Rate limiting types
export interface RateLimitInfo {
  limit: number;
  remaining: number;
  reset: number; // timestamp
  retryAfter?: number; // seconds
}

export interface RateLimitExceeded extends ApiError {
  rateLimitInfo: RateLimitInfo;
}

// Pagination for API responses
export interface ApiPaginationParams {
  page?: number;
  limit?: number;
  offset?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface ApiPaginatedResponse<T> extends ApiResponse {
  data?: {
    items: T[];
    pagination: {
      page: number;
      limit: number;
      total: number;
      totalPages: number;
      hasNext: boolean;
      hasPrev: boolean;
    };
  };
}
