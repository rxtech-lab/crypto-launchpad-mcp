/**
 * Example usage patterns for TanStack Query with the configured client
 * These examples show how to use the query client configuration in practice
 */

import { useQuery, useMutation } from "@tanstack/react-query";
import { queryKeys } from "./query-client";
import { useQueryClient } from "@/hooks/use-query-client";

// Example: Fetching user tokens
export function useTokensQuery(userId: string) {
  return useQuery({
    queryKey: queryKeys.tokens.list(userId),
    queryFn: async () => {
      const response = await fetch(`/api/tokens?userId=${userId}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch tokens: ${response.statusText}`);
      }
      return response.json();
    },
    enabled: !!userId, // Only run query if userId is provided
  });
}

// Example: Creating a new token with optimistic updates
export function useCreateTokenMutation(userId: string) {
  const { queryClient, invalidateTokens } = useQueryClient();

  return useMutation({
    mutationFn: async (tokenData: { name: string; scopes: string[] }) => {
      const response = await fetch("/api/tokens", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...tokenData, userId }),
      });

      if (!response.ok) {
        throw new Error(`Failed to create token: ${response.statusText}`);
      }

      return response.json();
    },
    // Optimistic update
    onMutate: async (newToken) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({
        queryKey: queryKeys.tokens.list(userId),
      });

      // Snapshot the previous value
      const previousTokens = queryClient.getQueryData(
        queryKeys.tokens.list(userId)
      );

      // Optimistically update to the new value
      queryClient.setQueryData(queryKeys.tokens.list(userId), (old: any) => {
        return old
          ? [...old, { ...newToken, id: "temp-id", createdAt: new Date() }]
          : [newToken];
      });

      // Return a context object with the snapshotted value
      return { previousTokens };
    },
    // If the mutation fails, use the context returned from onMutate to roll back
    onError: (err, newToken, context) => {
      if (context?.previousTokens) {
        queryClient.setQueryData(
          queryKeys.tokens.list(userId),
          context.previousTokens
        );
      }
    },
    // Always refetch after error or success
    onSettled: () => {
      invalidateTokens(userId);
    },
  });
}

// Example: Fetching user sessions
export function useSessionsQuery(userId: string) {
  return useQuery({
    queryKey: queryKeys.sessions.list(userId),
    queryFn: async () => {
      const response = await fetch(`/api/sessions?userId=${userId}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch sessions: ${response.statusText}`);
      }
      return response.json();
    },
    enabled: !!userId,
    // Refetch sessions more frequently since they can change often
    staleTime: 2 * 60 * 1000, // 2 minutes
  });
}

// Example: Deleting a session
export function useDeleteSessionMutation(userId: string) {
  const { invalidateSessions } = useQueryClient();

  return useMutation({
    mutationFn: async (sessionId: string) => {
      const response = await fetch(`/api/sessions/${sessionId}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        throw new Error(`Failed to delete session: ${response.statusText}`);
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate and refetch sessions after successful deletion
      invalidateSessions(userId);
    },
  });
}

// Example: Prefetching data
export function usePrefetchUserData(userId: string) {
  const { prefetchTokens, prefetchSessions } = useQueryClient();

  const prefetchData = async () => {
    // Prefetch both tokens and sessions
    await Promise.all([prefetchTokens(userId), prefetchSessions(userId)]);
  };

  return { prefetchData };
}
