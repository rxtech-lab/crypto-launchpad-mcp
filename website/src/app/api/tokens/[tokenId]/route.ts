import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { deleteJwtToken, getUserJwtTokens } from "@/lib/db/queries";

// DELETE /api/tokens/[tokenId] - Delete a JWT token
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ tokenId: string }> }
) {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
    }

    const { tokenId } = await params;

    // Verify the token belongs to the authenticated user
    const userTokens = await getUserJwtTokens(session.user.id);
    const tokenExists = userTokens.some((token) => token.id === tokenId);

    if (!tokenExists) {
      return NextResponse.json(
        { error: "Token not found or access denied" },
        { status: 404 }
      );
    }

    // Delete the token
    await deleteJwtToken(tokenId);

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Error deleting token:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 }
    );
  }
}
