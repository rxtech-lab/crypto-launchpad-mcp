import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { getUserSessions, deleteSession } from "@/lib/db/queries";

// POST /api/sessions/delete-others - Delete all sessions except current
export async function POST(request: NextRequest) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const body = await request.json();
    const { currentSessionToken } = body;

    if (!currentSessionToken) {
      return NextResponse.json(
        { error: "Current session token is required" },
        { status: 400 }
      );
    }

    // Get all user sessions
    const userSessions = await getUserSessions(session.user.id);

    // Delete all sessions except the current one
    const otherSessions = userSessions.filter(
      (s) => s.sessionToken !== currentSessionToken
    );

    for (const sessionToDelete of otherSessions) {
      await deleteSession(sessionToDelete.sessionToken);
    }

    return NextResponse.json({
      success: true,
      deletedCount: otherSessions.length,
    });
  } catch (error) {
    console.error("Error deleting other sessions:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
