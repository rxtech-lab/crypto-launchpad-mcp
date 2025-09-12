"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Session } from "@/lib/db/schema";

export interface SessionWithMetadata extends Session {
  userAgent: string;
  ipAddress: string;
}

export interface SessionsResponse {
  sessions: SessionWithMetadata[];
}

export interface DeleteOthersRequest {
  currentSessionToken: string;
}

export interface DeleteOthersResponse {
  success: boolean;
  deletedCount: number;
}

/**
 * Custom hook for session management with TanStack Query
 * Provides session operations with real-time updates and cache invalidation
 */
export function useSessions(userId?: string) {
  const queryClient = useQueryClient();

  // Fetch user's sessions
  const {
    data: sessionsData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["sessions", userId],
    queryFn: async (): Promise<SessionsResponse> => {
      if (!userId) {
        throw new Error("User ID is required");
      }

      const response = await fetch(`/api/sessions?userId=${userId}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to fetch sessions");
      }

      return response.json();
    },
    enabled: !!userId,
    staleTime: 1 * 60 * 1000, // 1 minute (sessions change more frequently)
    gcTime: 3 * 60 * 1000, // 3 minutes
    refetchInterval: 2 * 60 * 1000, // Refetch every 2 minutes for real-time updates
  });

  // Delete specific session mutation
  const deleteSessionMutation = useMutation({
    mutationFn: async (sessionToken: string): Promise<{ success: boolean }> => {
      const encodedToken = encodeURIComponent(sessionToken);
      const response = await fetch(`/api/sessions/${encodedToken}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to delete session");
      }

      return response.json();
    },
    onMutate: async (sessionToken) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ["sessions", userId] });

      // Snapshot the previous value
      const previousSessions = queryClient.getQueryData<SessionsResponse>([
        "sessions",
        userId,
      ]);

      // Optimistically update to the new value
      if (previousSessions) {
        queryClient.setQueryData<SessionsResponse>(["sessions", userId], {
          sessions: previousSessions.sessions.filter(
            (session) => session.sessionToken !== sessionToken
          ),
        });
      }

      return { previousSessions };
    },
    onError: (err, sessionToken, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousSessions) {
        queryClient.setQueryData(
          ["sessions", userId],
          context.previousSessions
        );
      }
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: ["sessions", userId] });
      // Also invalidate auth queries since session deletion affects authentication state
      queryClient.invalidateQueries({ queryKey: ["auth"] });
    },
  });

  // Delete all other sessions mutation
  const deleteOtherSessionsMutation = useMutation({
    mutationFn: async (
      data: DeleteOthersRequest
    ): Promise<DeleteOthersResponse> => {
      const response = await fetch("/api/sessions/delete-others", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to delete other sessions");
      }

      return response.json();
    },
    onMutate: async (data) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ["sessions", userId] });

      // Snapshot the previous value
      const previousSessions = queryClient.getQueryData<SessionsResponse>([
        "sessions",
        userId,
      ]);

      // Optimistically update to show only the current session
      if (previousSessions) {
        queryClient.setQueryData<SessionsResponse>(["sessions", userId], {
          sessions: previousSessions.sessions.filter(
            (session) => session.sessionToken === data.currentSessionToken
          ),
        });
      }

      return { previousSessions };
    },
    onError: (err, data, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousSessions) {
        queryClient.setQueryData(
          ["sessions", userId],
          context.previousSessions
        );
      }
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: ["sessions", userId] });
      // Also invalidate auth queries since session deletion affects authentication state
      queryClient.invalidateQueries({ queryKey: ["auth"] });
    },
  });

  // Helper function to get current session from the list
  const getCurrentSession = (currentSessionToken?: string) => {
    if (!currentSessionToken || !sessionsData?.sessions) {
      return null;
    }
    return (
      sessionsData.sessions.find(
        (session) => session.sessionToken === currentSessionToken
      ) || null
    );
  };

  // Helper function to get other sessions (excluding current)
  const getOtherSessions = (currentSessionToken?: string) => {
    if (!currentSessionToken || !sessionsData?.sessions) {
      return sessionsData?.sessions || [];
    }
    return sessionsData.sessions.filter(
      (session) => session.sessionToken !== currentSessionToken
    );
  };

  return {
    // Data
    sessions: sessionsData?.sessions || [],
    getCurrentSession,
    getOtherSessions,

    // Loading states
    isLoading,
    isDeleting: deleteSessionMutation.isPending,
    isDeletingOthers: deleteOtherSessionsMutation.isPending,

    // Error states
    error: error as Error | null,
    deleteError: deleteSessionMutation.error as Error | null,
    deleteOthersError: deleteOtherSessionsMutation.error as Error | null,

    // Actions
    deleteSession: deleteSessionMutation.mutate,
    deleteOtherSessions: deleteOtherSessionsMutation.mutate,
    refetch,

    // Mutation objects for additional control
    deleteSessionMutation,
    deleteOtherSessionsMutation,
  };
}
