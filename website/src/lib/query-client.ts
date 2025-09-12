import { QueryClient } from "@tanstack/react-query";

// Custom error class for API errors
export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public statusText: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// Create query client with comprehensive configuration
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Time before data is considered stale
        staleTime: 5 * 60 * 1000, // 5 minutes
        // Time before inactive queries are garbage collected
        gcTime: 10 * 60 * 1000, // 10 minutes
        // Retry configuration
        retry: (failureCount, error) => {
          // Don't retry on 4xx errors (client errors)
          if (
            error instanceof ApiError &&
            error.status >= 400 &&
            error.status < 500
          ) {
            return false;
          }
          // Don't retry on network errors in development
          if (
            process.env.NODE_ENV === "development" &&
            error instanceof TypeError
          ) {
            return false;
          }
          // Retry up to 3 times for other errors
          return failureCount < 3;
        },
        // Retry delay with exponential backoff
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
        // Refetch behavior
        refetchOnWindowFocus: process.env.NODE_ENV === "production",
        refetchOnReconnect: true,
        refetchInterval: false,
        // Network mode
        networkMode: "online",
      },
      mutations: {
        // Retry configuration for mutations
        retry: (failureCount, error) => {
          // Don't retry on 4xx errors (client errors)
          if (
            error instanceof ApiError &&
            error.status >= 400 &&
            error.status < 500
          ) {
            return false;
          }
          // Retry up to 2 times for server errors
          return failureCount < 2;
        },
        // Retry delay for mutations
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
        // Network mode
        networkMode: "online",
      },
    },
  });
}

// Query key factories for consistent key management
export const queryKeys = {
  // Authentication queries
  auth: {
    session: () => ["auth", "session"] as const,
    user: (userId: string) => ["auth", "user", userId] as const,
  },
  // Token management queries
  tokens: {
    all: () => ["tokens"] as const,
    list: (userId: string) => ["tokens", "list", userId] as const,
    detail: (tokenId: string) => ["tokens", "detail", tokenId] as const,
  },
  // Session management queries
  sessions: {
    all: () => ["sessions"] as const,
    list: (userId: string) => ["sessions", "list", userId] as const,
    detail: (sessionId: string) => ["sessions", "detail", sessionId] as const,
  },
} as const;

// Utility function to invalidate related queries
export function getInvalidationKeys(
  type: "tokens" | "sessions",
  userId?: string
) {
  switch (type) {
    case "tokens":
      return userId
        ? [queryKeys.tokens.list(userId)]
        : [queryKeys.tokens.all()];
    case "sessions":
      return userId
        ? [queryKeys.sessions.list(userId)]
        : [queryKeys.sessions.all()];
    default:
      return [];
  }
}
