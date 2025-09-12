// Central export file for all type definitions

// Authentication types
export type {
  AuthenticatedUser,
  AuthError,
  SignInResult,
  JwtTokenPayload,
  JwtTokenCreationResult,
  AuthProvider,
  WebAuthnCredential,
  SessionMetadata,
  EnhancedSession,
} from "./auth";

// Database types
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
  DatabaseResult,
  QueryOptions,
  UserSessionWithMetadata,
  JwtTokenWithUser,
  DatabaseStatus,
  ActionResult,
  TokenActionResult,
  SessionActionResult,
  PaginationOptions,
  PaginatedResult,
  TokenFilters,
  SessionFilters,
} from "./database";

// JWT types
export type {
  CreateJwtTokenInput,
  CreateJwtTokenResponse,
  JwtValidationResult,
  JwtClaims,
  JwtTokenMetadata,
  JwtTokenWithPayload,
  JwtGenerationOptions,
  JwtVerificationOptions,
  JwtErrorType,
  JwtError,
  JwtTokenUsage,
  JwtScope,
  JwtRole,
  JwtAuditLogEntry,
} from "./jwt";

// Validation types
export type {
  ValidationResult,
  ValidationError,
  FormValidationState,
  ValidationRule,
  FieldValidation,
  TokenFormValidation,
  SessionValidation,
  AuthValidation,
  ApiValidationSchema,
  ValidationMiddlewareResult,
} from "./validation";

// API types
export type {
  ApiResponse,
  ApiError,
  TokensApiResponse,
  CreateTokenApiRequest,
  CreateTokenApiResponse,
  DeleteTokenApiResponse,
  SessionsApiResponse,
  SessionWithMetadata,
  DeleteSessionApiResponse,
  DeleteOtherSessionsApiRequest,
  DeleteOtherSessionsApiResponse,
  AuthApiResponse,
  SignInApiRequest,
  SignInApiResponse,
  SignOutApiResponse,
  WebAuthnRegistrationRequest,
  WebAuthnRegistrationResponse,
  WebAuthnVerificationRequest,
  WebAuthnVerificationResponse,
  HttpMethod,
  ApiEndpoint,
  ApiRequestContext,
  ApiMiddleware,
  RateLimitInfo,
  RateLimitExceeded,
  ApiPaginationParams,
  ApiPaginatedResponse,
} from "./api";

// Quick Start types (existing)
export type * from "./quick-start";
