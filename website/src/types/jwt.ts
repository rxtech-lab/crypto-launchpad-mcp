// JWT Token specific type definitions

import type { AuthenticatedUser } from "./auth";
import type { JwtToken } from "./database";

// JWT Token creation input
export interface CreateJwtTokenInput {
  tokenName: string;
  aud?: string[];
  clientId?: string;
  roles?: string[];
  scopes?: string[];
  expiresIn?: string;
}

// JWT Token creation response
export interface CreateJwtTokenResponse {
  tokenRecord: JwtToken;
  jwt: string;
  authenticatedUser: AuthenticatedUser;
}

// JWT Token validation result
export interface JwtValidationResult {
  isValid: boolean;
  payload?: AuthenticatedUser;
  error?: string;
  reason?: "expired" | "invalid_signature" | "malformed" | "not_found";
}

// JWT Token claims (standard + custom)
export interface JwtClaims {
  // Standard JWT claims
  iss: string; // Issuer
  sub: string; // Subject (user ID)
  aud: string[]; // Audience
  exp: number; // Expiration time
  iat: number; // Issued at
  nbf: number; // Not before
  jti: string; // JWT ID

  // Custom claims
  client_id: string; // Client identifier
  oid: string; // Object ID (user ID)
  resid: string; // Resource ID (user ID)
  roles: string[]; // User roles
  scopes: string[]; // Token scopes
  sid: string; // Session ID
}

// JWT Token metadata for database storage
export interface JwtTokenMetadata {
  id: string;
  userId: string;
  tokenName: string;
  jti: string;
  aud: string[];
  clientId: string;
  roles: string[];
  scopes: string[];
  createdAt: Date;
  expiresAt: Date | null;
  isActive: boolean;
}

// JWT Token with decoded payload
export interface JwtTokenWithPayload extends JwtToken {
  decodedPayload?: AuthenticatedUser;
  isExpired?: boolean;
  timeUntilExpiry?: number; // seconds
}

// JWT Token generation options
export interface JwtGenerationOptions {
  algorithm?: "HS256" | "HS384" | "HS512" | "RS256" | "RS384" | "RS512";
  issuer?: string;
  audience?: string | string[];
  expiresIn?: string | number;
  notBefore?: string | number;
  subject?: string;
  jwtid?: string;
}

// JWT Token verification options
export interface JwtVerificationOptions {
  algorithms?: string[];
  audience?: string | string[];
  issuer?: string;
  ignoreExpiration?: boolean;
  ignoreNotBefore?: boolean;
  clockTolerance?: number;
  maxAge?: string | number;
}

// JWT Token error types
export type JwtErrorType =
  | "TokenExpiredError"
  | "JsonWebTokenError"
  | "NotBeforeError"
  | "TokenNotFoundError"
  | "TokenInactiveError"
  | "InvalidAudienceError"
  | "InvalidIssuerError";

export interface JwtError {
  type: JwtErrorType;
  message: string;
  expiredAt?: Date;
}

// JWT Token usage statistics (for future analytics)
export interface JwtTokenUsage {
  tokenId: string;
  jti: string;
  lastUsed?: Date;
  usageCount: number;
  ipAddresses: string[];
  userAgents: string[];
}

// JWT Token scope definitions
export interface JwtScope {
  name: string;
  description: string;
  permissions: string[];
}

// JWT Token role definitions
export interface JwtRole {
  name: string;
  description: string;
  scopes: string[];
  permissions: string[];
}

// JWT Token audit log entry
export interface JwtAuditLogEntry {
  id: string;
  tokenId: string;
  jti: string;
  action: "created" | "used" | "revoked" | "expired" | "deleted";
  timestamp: Date;
  userId: string;
  ipAddress?: string;
  userAgent?: string;
  metadata?: Record<string, any>;
}
