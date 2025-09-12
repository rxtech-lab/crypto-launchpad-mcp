"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { auth } from "@/auth";
import {
  createJwtToken,
  getUserJwtTokens,
  deactivateJwtToken,
  deleteJwtToken,
  getJwtTokenByJti,
} from "@/lib/db/queries";
import { generateJwtToken, getExpirationDate } from "@/lib/auth/jwt";
import { v4 as uuidv4 } from "uuid";
import type { JwtToken } from "@/lib/db/schema";
import type { JwtTokenPayload } from "@/types/auth";
import {
  validateCreateToken,
  validateTokenId,
  validateJti,
  type CreateTokenInput,
} from "@/lib/validation/token-validation";

export interface TokenActionResult {
  success: boolean;
  error?: string;
  data?: any;
}

/**
 * Create a new JWT token for the authenticated user
 */
export async function createToken(data: unknown): Promise<TokenActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate input data
    const validation = validateCreateToken(data);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const tokenData = validation.data!;

    // Prepare token payload with client ID
    const tokenPayload: JwtTokenPayload = {
      ...tokenData,
      clientId: tokenData.clientId || `client-${uuidv4()}`,
    };

    // Generate JWT token
    const { token: jwt, authenticatedUser } = generateJwtToken(
      session.user.id,
      tokenPayload
    );

    // Store token metadata in database
    const tokenRecord = await createJwtToken({
      id: uuidv4(),
      userId: session.user.id,
      tokenName: tokenData.tokenName,
      jti: authenticatedUser.jti,
      aud: authenticatedUser.aud,
      clientId: authenticatedUser.client_id,
      roles: authenticatedUser.roles,
      scopes: authenticatedUser.scopes,
      expiresAt: getExpirationDate(tokenData.expiresIn),
      isActive: true,
    });

    // Revalidate the tokens page to show the new token
    revalidatePath("/dashboard/tokens");

    return {
      success: true,
      data: {
        tokenRecord,
        jwt,
        authenticatedUser,
      },
    };
  } catch (error) {
    console.error("Error creating token:", error);
    return {
      success: false,
      error: "Failed to create token. Please try again.",
    };
  }
}

/**
 * Get all JWT tokens for the authenticated user
 */
export async function getTokens(): Promise<TokenActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    const tokens = await getUserJwtTokens(session.user.id);

    return {
      success: true,
      data: { tokens },
    };
  } catch (error) {
    console.error("Error fetching tokens:", error);
    return {
      success: false,
      error: "Failed to fetch tokens. Please try again.",
    };
  }
}

/**
 * Deactivate a JWT token (soft delete - marks as inactive)
 */
export async function deactivateToken(
  tokenId: unknown
): Promise<TokenActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate token ID
    const validation = validateTokenId(tokenId);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validTokenId = validation.data!;

    // Verify the token belongs to the authenticated user
    const tokens = await getUserJwtTokens(session.user.id);
    const tokenExists = tokens.some((token) => token.id === validTokenId);

    if (!tokenExists) {
      return {
        success: false,
        error: "Token not found or access denied",
      };
    }

    await deactivateJwtToken(validTokenId);

    // Revalidate the tokens page to reflect the change
    revalidatePath("/dashboard/tokens");

    return {
      success: true,
      data: { message: "Token deactivated successfully" },
    };
  } catch (error) {
    console.error("Error deactivating token:", error);
    return {
      success: false,
      error: "Failed to deactivate token. Please try again.",
    };
  }
}

/**
 * Permanently delete a JWT token
 */
export async function removeToken(
  tokenId: unknown
): Promise<TokenActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate token ID
    const validation = validateTokenId(tokenId);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validTokenId = validation.data!;

    // Verify the token belongs to the authenticated user
    const tokens = await getUserJwtTokens(session.user.id);
    const tokenExists = tokens.some((token) => token.id === validTokenId);

    if (!tokenExists) {
      return {
        success: false,
        error: "Token not found or access denied",
      };
    }

    await deleteJwtToken(validTokenId);

    // Revalidate the tokens page to reflect the change
    revalidatePath("/dashboard/tokens");

    return {
      success: true,
      data: { message: "Token deleted successfully" },
    };
  } catch (error) {
    console.error("Error deleting token:", error);
    return {
      success: false,
      error: "Failed to delete token. Please try again.",
    };
  }
}

/**
 * Get a specific JWT token by JTI (for validation purposes)
 */
export async function getTokenByJti(jti: unknown): Promise<TokenActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      redirect("/auth");
    }

    // Validate JTI
    const validation = validateJti(jti);
    if (!validation.success) {
      return {
        success: false,
        error: validation.error,
      };
    }

    const validJti = validation.data!;

    const token = await getJwtTokenByJti(validJti);

    if (!token) {
      return {
        success: false,
        error: "Token not found",
      };
    }

    // Verify the token belongs to the authenticated user
    if (token.userId !== session.user.id) {
      return {
        success: false,
        error: "Access denied",
      };
    }

    return {
      success: true,
      data: { token },
    };
  } catch (error) {
    console.error("Error fetching token by JTI:", error);
    return {
      success: false,
      error: "Failed to fetch token. Please try again.",
    };
  }
}
