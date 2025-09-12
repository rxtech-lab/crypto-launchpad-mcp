"use client";

import { useSession } from "next-auth/react";
import { useQuery } from "@tanstack/react-query";
import type { Session } from "next-auth";

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  image?: string;
}

export interface AuthState {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: Error | null;
  session: Session | null;
}

/**
 * Custom hook for authentication state management
 * Integrates with Auth.js session handling and provides authentication status
 */
export function useAuth(): AuthState {
  const { data: session, status } = useSession();

  // Use TanStack Query to manage authentication state with caching
  const {
    data: authData,
    isLoading: queryLoading,
    error,
  } = useQuery({
    queryKey: ["auth", session?.user?.id],
    queryFn: async () => {
      if (!session?.user) {
        return {
          user: null,
          isAuthenticated: false,
          session: null,
        };
      }

      const user: AuthUser = {
        id: session.user.id as string,
        email: session.user.email as string,
        name: session.user.name as string,
        image: session.user.image as string | undefined,
      };

      return {
        user,
        isAuthenticated: true,
        session,
      };
    },
    enabled: status !== "loading", // Only run query when session status is determined
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
  });

  const isLoading = status === "loading" || queryLoading;

  return {
    user: authData?.user || null,
    isAuthenticated: authData?.isAuthenticated || false,
    isLoading,
    error: error as Error | null,
    session: authData?.session || null,
  };
}
