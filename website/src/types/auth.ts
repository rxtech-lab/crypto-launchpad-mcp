import type { DefaultSession } from "next-auth";

declare module "next-auth" {
  interface Session {
    user: {
      id: string;
    } & DefaultSession["user"];
    sessionToken?: string;
  }
}

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

export interface AuthError {
  type: string;
  message: string;
}

export interface SignInResult {
  error?: string;
  status: number;
  ok: boolean;
  url?: string;
}

// JWT Token payload structure for token generation
export interface JwtTokenPayload {
  tokenName: string;
  aud: string[];
  clientId?: string;
  roles: string[];
  scopes: string[];
  expiresIn?: string; // e.g., "7d", "30d", "1y"
}

// JWT Token creation response
export interface JwtTokenCreationResult {
  token: string;
  authenticatedUser: AuthenticatedUser;
}

// Authentication provider types
export type AuthProvider = "webauthn" | "google";

// WebAuthn credential types
export interface WebAuthnCredential {
  id: string;
  rawId: ArrayBuffer;
  response: AuthenticatorAttestationResponse | AuthenticatorAssertionResponse;
  type: "public-key";
}

// Session metadata for enhanced session management
export interface SessionMetadata {
  userAgent?: string;
  ipAddress?: string;
  deviceType?: string;
  browser?: string;
  os?: string;
  location?: string;
  lastActivity?: Date;
}

// Enhanced session with metadata
export interface EnhancedSession {
  sessionToken: string;
  userId: string;
  expires: Date;
  metadata: SessionMetadata;
  isCurrent?: boolean;
}
