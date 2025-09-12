"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Copy,
  Eye,
  EyeOff,
  Trash2,
  Calendar,
  Key,
  AlertTriangle,
} from "lucide-react";
import type { JwtToken } from "@/lib/db/schema";

interface TokenManagerProps {
  tokens: JwtToken[];
  onDeleteToken: (tokenId: string) => Promise<void>;
  onCopyToken?: (token: string) => void;
}

interface TokenWithJwt extends JwtToken {
  jwt?: string;
}

export function TokenManager({
  tokens,
  onDeleteToken,
  onCopyToken,
}: TokenManagerProps) {
  const [visibleTokens, setVisibleTokens] = useState<Set<string>>(new Set());
  const [deletingTokens, setDeletingTokens] = useState<Set<string>>(new Set());

  const toggleTokenVisibility = (tokenId: string) => {
    setVisibleTokens((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(tokenId)) {
        newSet.delete(tokenId);
      } else {
        newSet.add(tokenId);
      }
      return newSet;
    });
  };

  const handleDeleteToken = async (tokenId: string) => {
    if (
      !confirm(
        "Are you sure you want to delete this token? This action cannot be undone."
      )
    ) {
      return;
    }

    setDeletingTokens((prev) => new Set(prev).add(tokenId));
    try {
      await onDeleteToken(tokenId);
    } finally {
      setDeletingTokens((prev) => {
        const newSet = new Set(prev);
        newSet.delete(tokenId);
        return newSet;
      });
    }
  };

  const handleCopyToken = (token: string) => {
    navigator.clipboard.writeText(token);
    if (onCopyToken) {
      onCopyToken(token);
    }
  };

  const formatDate = (date: Date | null) => {
    if (!date) return "Never";
    return new Intl.DateTimeFormat("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(new Date(date));
  };

  const isExpired = (expiresAt: Date | null) => {
    if (!expiresAt) return false;
    return new Date(expiresAt) < new Date();
  };

  if (tokens.length === 0) {
    return (
      <Card className="p-8 text-center">
        <Key className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">
          No JWT Tokens
        </h3>
        <p className="text-gray-600 mb-4">
          You haven't created any JWT tokens yet. Create your first token to get
          started.
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {tokens.map((token) => {
        const tokenWithJwt = token as TokenWithJwt;
        const isTokenVisible = visibleTokens.has(token.id);
        const isDeleting = deletingTokens.has(token.id);
        const expired = isExpired(token.expiresAt);

        return (
          <Card
            key={token.id}
            className={`p-6 ${expired ? "border-red-200 bg-red-50" : ""}`}
          >
            <div className="flex items-start justify-between mb-4">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="text-lg font-semibold text-gray-900">
                    {token.tokenName}
                  </h3>
                  <Badge
                    variant={
                      token.isActive && !expired ? "default" : "secondary"
                    }
                    className={expired ? "bg-red-100 text-red-800" : ""}
                  >
                    {expired
                      ? "Expired"
                      : token.isActive
                      ? "Active"
                      : "Inactive"}
                  </Badge>
                  {expired && (
                    <AlertTriangle className="h-4 w-4 text-red-500" />
                  )}
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-gray-600">
                  <div>
                    <span className="font-medium">Client ID:</span>{" "}
                    {token.clientId || "Not set"}
                  </div>
                  <div>
                    <span className="font-medium">JTI:</span> {token.jti}
                  </div>
                  <div className="flex items-center gap-1">
                    <Calendar className="h-4 w-4" />
                    <span className="font-medium">Created:</span>{" "}
                    {formatDate(token.createdAt)}
                  </div>
                  <div className="flex items-center gap-1">
                    <Calendar className="h-4 w-4" />
                    <span className="font-medium">Expires:</span>{" "}
                    {formatDate(token.expiresAt)}
                  </div>
                </div>

                {/* Audiences */}
                {token.aud && token.aud.length > 0 && (
                  <div className="mt-3">
                    <span className="text-sm font-medium text-gray-700">
                      Audiences:
                    </span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {token.aud.map((aud, index) => (
                        <Badge
                          key={index}
                          variant="outline"
                          className="text-xs"
                        >
                          {aud}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* Roles */}
                {token.roles && token.roles.length > 0 && (
                  <div className="mt-3">
                    <span className="text-sm font-medium text-gray-700">
                      Roles:
                    </span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {token.roles.map((role, index) => (
                        <Badge
                          key={index}
                          variant="outline"
                          className="text-xs"
                        >
                          {role}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* Scopes */}
                {token.scopes && token.scopes.length > 0 && (
                  <div className="mt-3">
                    <span className="text-sm font-medium text-gray-700">
                      Scopes:
                    </span>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {token.scopes.map((scope, index) => (
                        <Badge
                          key={index}
                          variant="outline"
                          className="text-xs"
                        >
                          {scope}
                        </Badge>
                      ))}
                    </div>
                  </div>
                )}

                {/* JWT Token Display */}
                {tokenWithJwt.jwt && (
                  <div className="mt-4">
                    <div className="flex items-center gap-2 mb-2">
                      <span className="text-sm font-medium text-gray-700">
                        JWT Token:
                      </span>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleTokenVisibility(token.id)}
                        className="h-6 px-2"
                      >
                        {isTokenVisible ? (
                          <EyeOff className="h-3 w-3" />
                        ) : (
                          <Eye className="h-3 w-3" />
                        )}
                      </Button>
                    </div>
                    <div className="bg-gray-100 rounded-md p-3 font-mono text-sm break-all">
                      {isTokenVisible ? (
                        <div className="flex items-center justify-between">
                          <span className="flex-1 mr-2">
                            {tokenWithJwt.jwt}
                          </span>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleCopyToken(tokenWithJwt.jwt!)}
                            className="h-6 px-2 flex-shrink-0"
                          >
                            <Copy className="h-3 w-3" />
                          </Button>
                        </div>
                      ) : (
                        <span className="text-gray-500">
                          ••••••••••••••••••••••••••••••••
                        </span>
                      )}
                    </div>
                  </div>
                )}
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2 ml-4">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteToken(token.id)}
                  disabled={isDeleting}
                  className="text-red-600 hover:text-red-700 hover:bg-red-50"
                >
                  {isDeleting ? (
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-red-600 border-t-transparent" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
          </Card>
        );
      })}
    </div>
  );
}
