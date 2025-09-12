"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { auth } from "@/auth";
import {
  getUserSessions,
  deleteSession,
  getSessionByToken,
} from "@/lib/db/queries";
import { eq, and, ne } from "drizzle-orm";
import { db } from "@/lib/db/connection";
import { sessions } from "@/lib/db/schema";
import type { Session } from "@/lib/db/schema";
import {
  validateSessionToken,
  validateSessionOperation,
} from "@/lib/validation/session-validation";

export interface SessionActionResult {
  success: boolean;
  error?: string;
  data?: any;
}

export interface SessionWithMetadata extends Session {
  userAgent?: string;
  ipAddress?: string;
  isCurrent?: boolean;
}

/**
 * Get all sessions for the authenticated user with metadata
 */
export async function getSessions(): Promise<SessionActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    const userSessions = await getUserSessions(session.user.id);

    // Add metadata to sessions (in a real app, you'd get this from request logs or session storage)
    // For now, we'll use placeholder data
    const sessionsWithMetadata: SessionWithMetadata[] = userSessions.map(
      (s) => ({
        ...s,
        userAgent:
          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", // Placeholder
        ipAddress: "192.168.1.1", // Placeholder
        isCurrent: s.sessionToken === session.sessionToken,
      })
    );

    return {
      success: true,
      data: { sessions: sessionsWithMetadata },
    };
  } catch (error) {
    console.error("Error fetching sessions:", error);
    return {
      success: false,
      error: "Failed to fetch sessions. Please try again.",
    };
  }
}

/**
 * Delete a specific session by session token
 */
export async function removeSession(
  sessionToken: unknown
): Promise<SessionActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate session token
    const validation = validateSessionToken(sessionToken);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validSessionToken = validation.data!;

    // Verify the session belongs to the authenticated user
    const targetSession = await getSessionByToken(validSessionToken);
    if (!targetSession) {
      return {
        success: false,
        error: "Session not found",
      };
    }

    if (targetSession.userId !== session.user.id) {
      return {
        success: false,
        error: "Access denied",
      };
    }

    // Validate session operation (prevent deleting current session)
    const operationValidation = validateSessionOperation(
      validSessionToken,
      session.sessionToken
    );
    if (!operationValidation.success) {
      return {
        success: false,
        error: operationValidation.error,
      };
    }

    await deleteSession(validSessionToken);

    // Revalidate the sessions page to reflect the change
    revalidatePath("/dashboard/sessions");

    return {
      success: true,
      data: { message: "Session deleted successfully" },
    };
  } catch (error) {
    console.error("Error deleting session:", error);
    return {
      success: false,
      error: "Failed to delete session. Please try again.",
    };
  }
}

/**
 * Delete all other sessions except the current one
 */
export async function removeOtherSessions(): Promise<SessionActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id || !session.sessionToken) {
      redirect("/auth");
    }

    // Delete all sessions for the user except the current one
    await db
      .delete(sessions)
      .where(
        and(
          eq(sessions.userId, session.user.id),
          ne(sessions.sessionToken, session.sessionToken)
        )
      );

    // Revalidate the sessions page to reflect the change
    revalidatePath("/dashboard/sessions");

    return {
      success: true,
      data: { message: "All other sessions deleted successfully" },
    };
  } catch (error) {
    console.error("Error deleting other sessions:", error);
    return {
      success: false,
      error: "Failed to delete other sessions. Please try again.",
    };
  }
}

/**
 * Get session details by session token
 */
export async function getSessionDetails(
  sessionToken: unknown
): Promise<SessionActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate session token
    const validation = validateSessionToken(sessionToken);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validSessionToken = validation.data!;

    const targetSession = await getSessionByToken(validSessionToken);
    if (!targetSession) {
      return {
        success: false,
        error: "Session not found",
      };
    }

    // Verify the session belongs to the authenticated user
    if (targetSession.userId !== session.user.id) {
      return {
        success: false,
        error: "Access denied",
      };
    }

    // Add metadata
    const sessionWithMetadata: SessionWithMetadata = {
      ...targetSession,
      userAgent:
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", // Placeholder
      ipAddress: "192.168.1.1", // Placeholder
      isCurrent: targetSession.sessionToken === session.sessionToken,
    };

    return {
      success: true,
      data: { session: sessionWithMetadata },
    };
  } catch (error) {
    console.error("Error fetching session details:", error);
    return {
      success: false,
      error: "Failed to fetch session details. Please try again.",
    };
  }
}

/**
 * Validate if a session token is valid and active
 */
export async function validateSession(
  sessionToken: unknown
): Promise<SessionActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate session token
    const validation = validateSessionToken(sessionToken);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validSessionToken = validation.data!;

    const targetSession = await getSessionByToken(validSessionToken);
    if (!targetSession) {
      return {
        success: false,
        data: { isValid: false, reason: "Session not found" },
      };
    }

    // Check if session has expired
    const now = new Date();
    const isExpired = targetSession.expires < now;

    if (isExpired) {
      return {
        success: true,
        data: { isValid: false, reason: "Session expired" },
      };
    }

    // Verify the session belongs to the authenticated user
    if (targetSession.userId !== session.user.id) {
      return {
        success: false,
        error: "Access denied",
      };
    }

    return {
      success: true,
      data: {
        isValid: true,
        session: targetSession,
        isCurrent: targetSession.sessionToken === session.sessionToken,
      },
    };
  } catch (error) {
    console.error("Error validating session:", error);
    return {
      success: false,
      error: "Failed to validate session. Please try again.",
    };
  }
}
