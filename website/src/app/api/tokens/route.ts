import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { generateJwtToken } from "@/lib/auth/jwt";
import { createJwtToken, getUserJwtTokens } from "@/lib/db/queries";
import { v4 as uuidv4 } from "uuid";
import { getExpirationDate } from "@/lib/auth/jwt";

// GET /api/tokens - Get user's JWT tokens
export async function GET(request: NextRequest) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const { searchParams } = new URL(request.url);
    const userId = searchParams.get("userId");

    // Verify the user is requesting their own tokens
    if (userId !== session.user.id) {
      return NextResponse.json({ error: "Forbidden" }, { status: 403 });
    }

    const tokens = await getUserJwtTokens(session.user.id);

    return NextResponse.json({ tokens });
  } catch (error) {
    console.error("Error fetching tokens:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}

// POST /api/tokens - Create a new JWT token
export async function POST(request: NextRequest) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const body = await request.json();
    const {
      userId,
      tokenName,
      aud = [],
      clientId = "",
      roles = [],
      scopes = [],
      expiresIn = "30d",
    } = body;

    // Verify the user is creating a token for themselves
    if (userId !== session.user.id) {
      return NextResponse.json({ error: "Forbidden" }, { status: 403 });
    }

    // Validate required fields
    if (!tokenName?.trim()) {
      return NextResponse.json(
        { error: "Token name is required" },
        { status: 400 }
      );
    }

    // Generate JWT token
    const { token: jwt, authenticatedUser } = generateJwtToken(userId, {
      tokenName: tokenName.trim(),
      aud: Array.isArray(aud) ? aud : [],
      clientId: clientId || `client-${uuidv4()}`,
      roles: Array.isArray(roles) ? roles : [],
      scopes: Array.isArray(scopes) ? scopes : [],
      expiresIn,
    });

    // Store token metadata in database
    const tokenRecord = await createJwtToken({
      id: uuidv4(),
      userId,
      tokenName: tokenName.trim(),
      jti: authenticatedUser.jti,
      aud: authenticatedUser.aud,
      clientId: authenticatedUser.client_id,
      roles: authenticatedUser.roles,
      scopes: authenticatedUser.scopes,
      expiresAt: getExpirationDate(expiresIn),
      isActive: true,
    });

    return NextResponse.json({
      tokenRecord,
      jwt,
      authenticatedUser,
    });
  } catch (error) {
    console.error("Error creating token:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
