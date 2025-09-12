"use client";

import { useState, useEffect } from "react";
import { useSession } from "next-auth/react";
import { CreateTokenForm } from "@/components/dashboard/create-token-form";
import { TokenManager } from "@/components/dashboard/token-manager";
import type { JwtToken } from "@/lib/db/schema";

interface TokenWithJwt extends JwtToken {
  jwt?: string;
}

export default function TokensPage() {
  const { data: session } = useSession();
  const [tokens, setTokens] = useState<TokenWithJwt[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isCreating, setIsCreating] = useState(false);

  // Load tokens on component mount
  useEffect(() => {
    if (session?.user?.id) {
      loadTokens();
    }
  }, [session?.user?.id]);

  const loadTokens = async () => {
    if (!session?.user?.id) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/tokens?userId=${session.user.id}`);
      if (response.ok) {
        const data = await response.json();
        setTokens(data.tokens || []);
      } else {
        console.error("Failed to load tokens");
      }
    } catch (error) {
      console.error("Error loading tokens:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateToken = async (tokenData: {
    tokenName: string;
    aud: string[];
    clientId: string;
    roles: string[];
    scopes: string[];
    expiresIn: string;
  }) => {
    if (!session?.user?.id) return;

    setIsCreating(true);
    try {
      const response = await fetch("/api/tokens", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          userId: session.user.id,
          ...tokenData,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        // Add the new token with JWT to the list
        setTokens((prev) => [{ ...data.tokenRecord, jwt: data.jwt }, ...prev]);
      } else {
        const error = await response.json();
        alert(`Failed to create token: ${error.message}`);
      }
    } catch (error) {
      console.error("Error creating token:", error);
      alert("Failed to create token. Please try again.");
    } finally {
      setIsCreating(false);
    }
  };

  const handleDeleteToken = async (tokenId: string) => {
    try {
      const response = await fetch(`/api/tokens/${tokenId}`, {
        method: "DELETE",
      });

      if (response.ok) {
        setTokens((prev) => prev.filter((token) => token.id !== tokenId));
      } else {
        const error = await response.json();
        alert(`Failed to delete token: ${error.message}`);
      }
    } catch (error) {
      console.error("Error deleting token:", error);
      alert("Failed to delete token. Please try again.");
    }
  };

  const handleCopyToken = (token: string) => {
    // Show a temporary success message
    const originalTitle = document.title;
    document.title = "Token copied!";
    setTimeout(() => {
      document.title = originalTitle;
    }, 2000);
  };

  if (!session) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-center">
          <p className="text-gray-600">Please sign in to manage your tokens.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          JWT Token Management
        </h1>
        <p className="text-gray-600">
          Create and manage JWT tokens for API access. These tokens can be used
          to authenticate with external services.
        </p>
      </div>

      {/* Create Token Form */}
      <CreateTokenForm onSubmit={handleCreateToken} isLoading={isCreating} />

      {/* Existing Tokens */}
      <div>
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900">Your Tokens</h2>
          {isLoading && (
            <div className="flex items-center gap-2 text-gray-600">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-gray-600 border-t-transparent" />
              <span className="text-sm">Loading...</span>
            </div>
          )}
        </div>

        <TokenManager
          tokens={tokens}
          onDeleteToken={handleDeleteToken}
          onCopyToken={handleCopyToken}
        />
      </div>
    </div>
  );
}
