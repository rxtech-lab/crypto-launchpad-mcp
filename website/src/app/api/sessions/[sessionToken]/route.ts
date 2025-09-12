import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { deleteSession, getUserSessions } from "@/lib/db/queries";

// DELETE /api/sessions/[sessionToken] - Delete a session
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ sessionToken: string }> }
) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const { sessionToken } = await params;
    const decodedSessionToken = decodeURIComponent(sessionToken);

    // Verify the session belongs to the authenticated user
    const userSessions = await getUserSessions(session.user.id);
    const sessionExists = userSessions.some(
      (s) => s.sessionToken === decodedSessionToken
    );

    if (!sessionExists) {
      return NextResponse.json(
        { error: "Session not found or access denied" },
        { status: 404 }
      );
    }

    // Delete the session
    await deleteSession(decodedSessionToken);

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Error deleting session:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
