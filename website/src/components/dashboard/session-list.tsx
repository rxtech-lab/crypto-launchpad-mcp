"use client";

import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Trash2,
  Calendar,
  Monitor,
  Smartphone,
  Globe,
  AlertTriangle,
  Shield,
} from "lucide-react";
import type { Session } from "@/lib/db/schema";

interface SessionWithMetadata extends Session {
  userAgent?: string;
  ipAddress?: string;
  isCurrent?: boolean;
}

interface SessionListProps {
  sessions: SessionWithMetadata[];
  onDeleteSession: (sessionToken: string) => Promise<void>;
  currentSessionToken?: string;
}

export function SessionList({
  sessions,
  onDeleteSession,
  currentSessionToken,
}: SessionListProps) {
  const [deletingSessions, setDeletingSessions] = useState<Set<string>>(
    new Set()
  );

  const handleDeleteSession = async (sessionToken: string) => {
    if (sessionToken === currentSessionToken) {
      if (
        !confirm(
          "This will sign you out of this session. Are you sure you want to continue?"
        )
      ) {
        return;
      }
    } else {
      if (
        !confirm(
          "Are you sure you want to delete this session? The user will be signed out."
        )
      ) {
        return;
      }
    }

    setDeletingSessions((prev) => new Set(prev).add(sessionToken));
    try {
      await onDeleteSession(sessionToken);
    } finally {
      setDeletingSessions((prev) => {
        const newSet = new Set(prev);
        newSet.delete(sessionToken);
        return newSet;
      });
    }
  };

  const formatDate = (date: Date | null) => {
    if (!date) return "Unknown";
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

  const getDeviceIcon = (userAgent?: string) => {
    if (!userAgent) return <Monitor className="h-4 w-4" />;

    const ua = userAgent.toLowerCase();
    if (
      ua.includes("mobile") ||
      ua.includes("android") ||
      ua.includes("iphone")
    ) {
      return <Smartphone className="h-4 w-4" />;
    }
    return <Monitor className="h-4 w-4" />;
  };

  const getBrowserInfo = (userAgent?: string) => {
    if (!userAgent) return "Unknown Browser";

    const ua = userAgent.toLowerCase();
    if (ua.includes("chrome")) return "Chrome";
    if (ua.includes("firefox")) return "Firefox";
    if (ua.includes("safari") && !ua.includes("chrome")) return "Safari";
    if (ua.includes("edge")) return "Edge";
    return "Unknown Browser";
  };

  if (sessions.length === 0) {
    return (
      <Card className="p-8 text-center">
        <Shield className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">
          No Active Sessions
        </h3>
        <p className="text-gray-600">
          There are no active sessions to display.
        </p>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {sessions.map((session) => {
        const isDeleting = deletingSessions.has(session.sessionToken);
        const expired = isExpired(session.expires);
        const isCurrent = session.sessionToken === currentSessionToken;

        return (
          <Card
            key={session.sessionToken}
            className={`p-6 ${expired ? "border-red-200 bg-red-50" : ""} ${
              isCurrent ? "border-blue-200 bg-blue-50" : ""
            }`}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-3">
                  {getDeviceIcon(session.userAgent)}
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900">
                      {getBrowserInfo(session.userAgent)}
                    </h3>
                    <div className="flex items-center gap-2 mt-1">
                      {isCurrent && (
                        <Badge variant="default" className="text-xs">
                          Current Session
                        </Badge>
                      )}
                      <Badge
                        variant={expired ? "secondary" : "default"}
                        className={
                          expired
                            ? "bg-red-100 text-red-800"
                            : "bg-green-100 text-green-800"
                        }
                      >
                        {expired ? "Expired" : "Active"}
                      </Badge>
                      {expired && (
                        <AlertTriangle className="h-4 w-4 text-red-500" />
                      )}
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm text-gray-600">
                  <div className="flex items-center gap-2">
                    <Calendar className="h-4 w-4" />
                    <span className="font-medium">Expires:</span>{" "}
                    {formatDate(session.expires)}
                  </div>

                  {session.ipAddress && (
                    <div className="flex items-center gap-2">
                      <Globe className="h-4 w-4" />
                      <span className="font-medium">IP Address:</span>{" "}
                      {session.ipAddress}
                    </div>
                  )}
                </div>

                {session.userAgent && (
                  <div className="mt-3 text-sm text-gray-600">
                    <span className="font-medium">User Agent:</span>
                    <div className="mt-1 p-2 bg-gray-100 rounded text-xs font-mono break-all">
                      {session.userAgent}
                    </div>
                  </div>
                )}

                <div className="mt-3 text-xs text-gray-500">
                  <span className="font-medium">Session Token:</span>{" "}
                  {session.sessionToken.slice(0, 16)}...
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2 ml-4">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDeleteSession(session.sessionToken)}
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
