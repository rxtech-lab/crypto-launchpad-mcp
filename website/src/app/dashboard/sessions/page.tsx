"use client";

import { useState, useEffect } from "react";
import { useSession } from "next-auth/react";
import { SessionList } from "@/components/dashboard/session-list";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { RefreshCw, AlertTriangle } from "lucide-react";
import type { Session } from "@/lib/db/schema";

interface SessionWithMetadata extends Session {
  userAgent?: string;
  ipAddress?: string;
  isCurrent?: boolean;
}

export default function SessionsPage() {
  const { data: session } = useSession();
  const [sessions, setSessions] = useState<SessionWithMetadata[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [currentSessionToken, setCurrentSessionToken] = useState<string>();

  // Load sessions on component mount
  useEffect(() => {
    if (session?.user?.id) {
      loadSessions();
      // Get current session token from cookies or session
      getCurrentSessionToken();
    }
  }, [session?.user?.id]);

  const getCurrentSessionToken = () => {
    // In a real implementation, you'd get this from the session or cookies
    // For now, we'll use a placeholder
    const cookies = document.cookie.split(";");
    const sessionCookie = cookies.find(
      (cookie) =>
        cookie.trim().startsWith("next-auth.session-token=") ||
        cookie.trim().startsWith("__Secure-next-auth.session-token=")
    );

    if (sessionCookie) {
      const token = sessionCookie.split("=")[1];
      setCurrentSessionToken(token);
    }
  };

  const loadSessions = async () => {
    if (!session?.user?.id) return;

    setIsLoading(true);
    try {
      const response = await fetch(`/api/sessions?userId=${session.user.id}`);
      if (response.ok) {
        const data = await response.json();
        setSessions(data.sessions || []);
      } else {
        console.error("Failed to load sessions");
      }
    } catch (error) {
      console.error("Error loading sessions:", error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteSession = async (sessionToken: string) => {
    try {
      const response = await fetch(
        `/api/sessions/${encodeURIComponent(sessionToken)}`,
        {
          method: "DELETE",
        }
      );

      if (response.ok) {
        setSessions((prev) =>
          prev.filter((s) => s.sessionToken !== sessionToken)
        );

        // If the user deleted their current session, they'll be redirected to login
        if (sessionToken === currentSessionToken) {
          window.location.href = "/auth";
        }
      } else {
        const error = await response.json();
        alert(`Failed to delete session: ${error.message}`);
      }
    } catch (error) {
      console.error("Error deleting session:", error);
      alert("Failed to delete session. Please try again.");
    }
  };

  const handleDeleteAllOtherSessions = async () => {
    if (
      !confirm(
        "Are you sure you want to sign out all other sessions? This will not affect your current session."
      )
    ) {
      return;
    }

    try {
      const response = await fetch("/api/sessions/delete-others", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          currentSessionToken,
        }),
      });

      if (response.ok) {
        // Reload sessions to reflect changes
        await loadSessions();
      } else {
        const error = await response.json();
        alert(`Failed to delete other sessions: ${error.message}`);
      }
    } catch (error) {
      console.error("Error deleting other sessions:", error);
      alert("Failed to delete other sessions. Please try again.");
    }
  };

  if (!session) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="text-center">
          <p className="text-gray-600">
            Please sign in to manage your sessions.
          </p>
        </div>
      </div>
    );
  }

  const activeSessions = sessions.filter(
    (s) => new Date(s.expires) > new Date()
  );
  const expiredSessions = sessions.filter(
    (s) => new Date(s.expires) <= new Date()
  );

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          Session Management
        </h1>
        <p className="text-gray-600">
          View and manage your active sessions. You can sign out of individual
          sessions or all other sessions for security.
        </p>
      </div>

      {/* Session Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">
                Active Sessions
              </p>
              <p className="text-2xl font-bold text-gray-900">
                {activeSessions.length}
              </p>
            </div>
          </div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">
                Expired Sessions
              </p>
              <p className="text-2xl font-bold text-gray-900">
                {expiredSessions.length}
              </p>
            </div>
          </div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">
                Total Sessions
              </p>
              <p className="text-2xl font-bold text-gray-900">
                {sessions.length}
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Actions */}
      <Card className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 mb-1">
              Session Actions
            </h2>
            <p className="text-sm text-gray-600">
              Manage your sessions for better security
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              onClick={loadSessions}
              disabled={isLoading}
              className="flex items-center gap-2"
            >
              <RefreshCw
                className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`}
              />
              Refresh
            </Button>

            {activeSessions.length > 1 && (
              <Button
                variant="outline"
                onClick={handleDeleteAllOtherSessions}
                className="flex items-center gap-2 text-red-600 hover:text-red-700 hover:bg-red-50"
              >
                <AlertTriangle className="h-4 w-4" />
                Sign Out Other Sessions
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* Active Sessions */}
      {activeSessions.length > 0 && (
        <div>
          <h2 className="text-xl font-semibold text-gray-900 mb-4">
            Active Sessions
          </h2>
          <SessionList
            sessions={activeSessions}
            onDeleteSession={handleDeleteSession}
            currentSessionToken={currentSessionToken}
          />
        </div>
      )}

      {/* Expired Sessions */}
      {expiredSessions.length > 0 && (
        <div>
          <h2 className="text-xl font-semibold text-gray-900 mb-4">
            Expired Sessions
          </h2>
          <SessionList
            sessions={expiredSessions}
            onDeleteSession={handleDeleteSession}
            currentSessionToken={currentSessionToken}
          />
        </div>
      )}

      {/* No Sessions */}
      {sessions.length === 0 && !isLoading && (
        <Card className="p-8 text-center">
          <p className="text-gray-600">No sessions found.</p>
        </Card>
      )}
    </div>
  );
}
