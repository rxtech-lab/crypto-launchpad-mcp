# TanStack Query Setup

This document describes the TanStack Query configuration and usage patterns implemented for the authentication dashboard.

## Configuration

### Query Client Setup

The query client is configured in `src/lib/query-client.ts` with the following features:

- **Stale Time**: 5 minutes (data is considered fresh for 5 minutes)
- **Garbage Collection Time**: 10 minutes (inactive queries are cleaned up after 10 minutes)
- **Retry Logic**: Smart retry with exponential backoff
  - No retry on 4xx client errors
  - No retry on network errors in development
  - Up to 3 retries for queries, 2 for mutations
- **Refetch Behavior**:
  - Refetch on window focus in production only
  - Refetch on reconnect
  - No automatic background refetching

### Provider Integration

The query client is integrated into the app through `src/components/providers.tsx`:

```tsx
import { createQueryClient } from "@/lib/query-client";

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => createQueryClient());

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        {children}
        {/* DevTools only in development */}
        {process.env.NODE_ENV === "development" && (
          <ReactQueryDevtools
            initialIsOpen={false}
            buttonPosition="bottom-left"
            position="bottom"
          />
        )}
      </SessionProvider>
    </QueryClientProvider>
  );
}
```

## Query Keys

Consistent query key management is provided through the `queryKeys` factory:

```typescript
export const queryKeys = {
  auth: {
    session: () => ["auth", "session"] as const,
    user: (userId: string) => ["auth", "user", userId] as const,
  },
  tokens: {
    all: () => ["tokens"] as const,
    list: (userId: string) => ["tokens", "list", userId] as const,
    detail: (tokenId: string) => ["tokens", "detail", tokenId] as const,
  },
  sessions: {
    all: () => ["sessions"] as const,
    list: (userId: string) => ["sessions", "list", userId] as const,
    detail: (sessionId: string) => ["sessions", "detail", sessionId] as const,
  },
} as const;
```

## Custom Hooks

### useQueryClient Hook

The `src/hooks/use-query-client.ts` provides utility methods for common operations:

```typescript
const { queryClient, invalidateTokens, prefetchTokens } = useQueryClient();

// Invalidate specific queries
await invalidateTokens(userId);
await invalidateSessions(userId);

// Prefetch data
await prefetchTokens(userId);
await prefetchSessions(userId);

// Direct cache manipulation
setTokensData(userId, newData);
const cachedData = getTokensData(userId);
```

## Error Handling

### ApiError Class

Custom error class for API responses:

```typescript
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
```

### Error Utilities

The `src/lib/query-utils.ts` provides error handling utilities:

```typescript
import {
  getErrorMessage,
  isApiError,
  getLoadingState,
} from "@/lib/query-utils";

// Extract user-friendly error messages
const errorMessage = getErrorMessage(error);

// Check error types
if (isApiError(error)) {
  console.log(`API Error: ${error.status}`);
}

// Get loading state for UI
const { isLoading, isError, error } = getLoadingState(query);
```

## Usage Examples

### Basic Query

```typescript
import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/lib/query-client";

function TokensList({ userId }: { userId: string }) {
  const {
    data: tokens,
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.tokens.list(userId),
    queryFn: async () => {
      const response = await fetch(`/api/tokens?userId=${userId}`);
      if (!response.ok) throw new Error("Failed to fetch tokens");
      return response.json();
    },
    enabled: !!userId,
  });

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      {tokens?.map((token) => (
        <div key={token.id}>{token.name}</div>
      ))}
    </div>
  );
}
```

### Mutation with Optimistic Updates

```typescript
import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@/hooks/use-query-client";

function CreateTokenForm({ userId }: { userId: string }) {
  const { queryClient, invalidateTokens } = useQueryClient();

  const createToken = useMutation({
    mutationFn: async (tokenData) => {
      const response = await fetch("/api/tokens", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...tokenData, userId }),
      });
      if (!response.ok) throw new Error("Failed to create token");
      return response.json();
    },
    onMutate: async (newToken) => {
      // Optimistic update
      await queryClient.cancelQueries({
        queryKey: queryKeys.tokens.list(userId),
      });
      const previousTokens = queryClient.getQueryData(
        queryKeys.tokens.list(userId)
      );

      queryClient.setQueryData(queryKeys.tokens.list(userId), (old: any) =>
        old ? [...old, { ...newToken, id: "temp-id" }] : [newToken]
      );

      return { previousTokens };
    },
    onError: (err, newToken, context) => {
      if (context?.previousTokens) {
        queryClient.setQueryData(
          queryKeys.tokens.list(userId),
          context.previousTokens
        );
      }
    },
    onSettled: () => {
      invalidateTokens(userId);
    },
  });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        createToken.mutate({ name: "New Token", scopes: ["read"] });
      }}
    >
      <button type="submit" disabled={createToken.isPending}>
        {createToken.isPending ? "Creating..." : "Create Token"}
      </button>
    </form>
  );
}
```

## DevTools

React Query DevTools are automatically included in development mode:

- **Position**: Bottom left corner
- **Initial State**: Closed
- **Access**: Click the floating button to open/close

The DevTools provide:

- Query inspection and debugging
- Cache visualization
- Network request monitoring
- Performance metrics
- Manual query triggering

## Best Practices

1. **Use Query Keys Factory**: Always use the `queryKeys` factory for consistent key management
2. **Handle Loading States**: Use the utility functions to handle loading and error states
3. **Implement Optimistic Updates**: For better UX, implement optimistic updates for mutations
4. **Cache Invalidation**: Use the custom hook utilities for proper cache invalidation
5. **Error Boundaries**: Implement error boundaries to catch and handle query errors gracefully
6. **Prefetching**: Use prefetching for data that users are likely to need soon

## Integration with Auth Dashboard

The TanStack Query setup is specifically configured to work with:

- **Authentication State**: Session and user data management
- **JWT Token Management**: CRUD operations for user tokens
- **Session Management**: Active session tracking and management
- **Real-time Updates**: Optimistic updates and cache invalidation for responsive UI

This configuration provides a robust foundation for the authentication dashboard's data management needs.
