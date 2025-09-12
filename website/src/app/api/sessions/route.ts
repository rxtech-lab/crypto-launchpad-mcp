import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { getUserSessions } from "@/lib/db/queries";

// GET /api/sessions - Get user's sessions
export async function GET(request: NextRequest) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const { searchParams } = new URL(request.url);
    const userId = searchParams.get("userId");

    // Verify the user is requesting their own sessions
    if (userId !== session.user.id) {
      return NextResponse.json({ error: "Forbidden" }, { status: 403 });
    }

    const sessions = await getUserSessions(session.user.id);

    // Add metadata to sessions (in a real app, you'd get this from request logs or session storage)
    const sessionsWithMetadata = sessions.map((s) => ({
      ...s,
      userAgent:
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36", // Placeholder
      ipAddress: "192.168.1.1", // Placeholder
    }));

    return NextResponse.json({ sessions: sessionsWithMetadata });
  } catch (error) {
    console.error("Error fetching sessions:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
