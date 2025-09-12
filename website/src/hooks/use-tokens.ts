"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { JwtToken } from "@/lib/db/schema";

export interface CreateTokenRequest {
  userId: string;
  tokenName: string;
  aud?: string[];
  clientId?: string;
  roles?: string[];
  scopes?: string[];
  expiresIn?: string;
}

export interface CreateTokenResponse {
  tokenRecord: JwtToken;
  jwt: string;
  authenticatedUser: {
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
  };
}

export interface TokensResponse {
  tokens: JwtToken[];
}

/**
 * Custom hook for JWT token management with TanStack Query
 * Provides CRUD operations with optimistic updates and cache management
 */
export function useTokens(userId?: string) {
  const queryClient = useQueryClient();

  // Fetch user's JWT tokens
  const {
    data: tokensData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["tokens", userId],
    queryFn: async (): Promise<TokensResponse> => {
      if (!userId) {
        throw new Error("User ID is required");
      }

      const response = await fetch(`/api/tokens?userId=${userId}`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to fetch tokens");
      }

      return response.json();
    },
    enabled: !!userId,
    staleTime: 2 * 60 * 1000, // 2 minutes
    gcTime: 5 * 60 * 1000, // 5 minutes
  });

  // Create new JWT token mutation
  const createTokenMutation = useMutation({
    mutationFn: async (
      tokenData: CreateTokenRequest
    ): Promise<CreateTokenResponse> => {
      const response = await fetch("/api/tokens", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(tokenData),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to create token");
      }

      return response.json();
    },
    onMutate: async (newToken) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ["tokens", userId] });

      // Snapshot the previous value
      const previousTokens = queryClient.getQueryData<TokensResponse>([
        "tokens",
        userId,
      ]);

      // Optimistically update to the new value
      if (previousTokens) {
        const optimisticToken: JwtToken = {
          id: `temp-${Date.now()}`,
          userId: newToken.userId,
          tokenName: newToken.tokenName,
          jti: `temp-jti-${Date.now()}`,
          aud: newToken.aud || [],
          clientId: newToken.clientId || "",
          roles: newToken.roles || [],
          scopes: newToken.scopes || [],
          createdAt: new Date(),
          expiresAt: null,
          isActive: true,
        };

        queryClient.setQueryData<TokensResponse>(["tokens", userId], {
          tokens: [...previousTokens.tokens, optimisticToken],
        });
      }

      return { previousTokens };
    },
    onError: (err, newToken, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousTokens) {
        queryClient.setQueryData(["tokens", userId], context.previousTokens);
      }
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: ["tokens", userId] });
    },
  });

  // Delete JWT token mutation
  const deleteTokenMutation = useMutation({
    mutationFn: async (tokenId: string): Promise<{ success: boolean }> => {
      const response = await fetch(`/api/tokens/${tokenId}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || "Failed to delete token");
      }

      return response.json();
    },
    onMutate: async (tokenId) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ["tokens", userId] });

      // Snapshot the previous value
      const previousTokens = queryClient.getQueryData<TokensResponse>([
        "tokens",
        userId,
      ]);

      // Optimistically update to the new value
      if (previousTokens) {
        queryClient.setQueryData<TokensResponse>(["tokens", userId], {
          tokens: previousTokens.tokens.filter((token) => token.id !== tokenId),
        });
      }

      return { previousTokens };
    },
    onError: (err, tokenId, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousTokens) {
        queryClient.setQueryData(["tokens", userId], context.previousTokens);
      }
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: ["tokens", userId] });
    },
  });

  return {
    // Data
    tokens: tokensData?.tokens || [],

    // Loading states
    isLoading,
    isCreating: createTokenMutation.isPending,
    isDeleting: deleteTokenMutation.isPending,

    // Error states
    error: error as Error | null,
    createError: createTokenMutation.error as Error | null,
    deleteError: deleteTokenMutation.error as Error | null,

    // Actions
    createToken: createTokenMutation.mutate,
    deleteToken: deleteTokenMutation.mutate,
    refetch,

    // Mutation objects for additional control
    createTokenMutation,
    deleteTokenMutation,
  };
}
