import { useQueryClient as useTanStackQueryClient } from "@tanstack/react-query";
import { queryKeys, getInvalidationKeys } from "@/lib/query-client";

/**
 * Custom hook that provides access to the query client with utility methods
 * for common operations like cache invalidation and prefetching
 */
export function useQueryClient() {
  const queryClient = useTanStackQueryClient();

  // Utility methods for common operations
  const utils = {
    // Invalidate queries by type
    invalidateTokens: (userId?: string) => {
      const keys = getInvalidationKeys("tokens", userId);
      return Promise.all(
        keys.map((key) => queryClient.invalidateQueries({ queryKey: key }))
      );
    },

    invalidateSessions: (userId?: string) => {
      const keys = getInvalidationKeys("sessions", userId);
      return Promise.all(
        keys.map((key) => queryClient.invalidateQueries({ queryKey: key }))
      );
    },

    invalidateAuth: () => {
      return queryClient.invalidateQueries({
        queryKey: queryKeys.auth.session(),
      });
    },

    // Prefetch queries
    prefetchTokens: (userId: string) => {
      return queryClient.prefetchQuery({
        queryKey: queryKeys.tokens.list(userId),
        staleTime: 5 * 60 * 1000, // 5 minutes
      });
    },

    prefetchSessions: (userId: string) => {
      return queryClient.prefetchQuery({
        queryKey: queryKeys.sessions.list(userId),
        staleTime: 5 * 60 * 1000, // 5 minutes
      });
    },

    // Remove queries from cache
    removeTokenQueries: (userId?: string) => {
      if (userId) {
        queryClient.removeQueries({ queryKey: queryKeys.tokens.list(userId) });
      } else {
        queryClient.removeQueries({ queryKey: queryKeys.tokens.all() });
      }
    },

    removeSessionQueries: (userId?: string) => {
      if (userId) {
        queryClient.removeQueries({
          queryKey: queryKeys.sessions.list(userId),
        });
      } else {
        queryClient.removeQueries({ queryKey: queryKeys.sessions.all() });
      }
    },

    // Set query data directly (for optimistic updates)
    setTokensData: <T>(userId: string, data: T) => {
      queryClient.setQueryData(queryKeys.tokens.list(userId), data);
    },

    setSessionsData: <T>(userId: string, data: T) => {
      queryClient.setQueryData(queryKeys.sessions.list(userId), data);
    },

    // Get cached data
    getTokensData: <T>(userId: string): T | undefined => {
      return queryClient.getQueryData(queryKeys.tokens.list(userId));
    },

    getSessionsData: <T>(userId: string): T | undefined => {
      return queryClient.getQueryData(queryKeys.sessions.list(userId));
    },

    // Clear all cache
    clearCache: () => {
      queryClient.clear();
    },

    // Reset queries to initial state
    resetQueries: () => {
      return queryClient.resetQueries();
    },
  };

  return {
    queryClient,
    ...utils,
  };
}

// Export query keys for use in components
export { queryKeys } from "@/lib/query-client";
