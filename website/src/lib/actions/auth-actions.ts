"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { auth, signOut } from "@/auth";
import { getUserById } from "@/lib/db/queries";
import type { User } from "@/lib/db/schema";

export interface AuthActionResult {
  success: boolean;
  error?: string;
  data?: any;
}

/**
 * Get the current authenticated user's profile information
 */
export async function getCurrentUser(): Promise<AuthActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return {
        success: false,
        error: "Not authenticated",
      };
    }

    const user = await getUserById(session.user.id);
    if (!user) {
      return {
        success: false,
        error: "User not found",
      };
    }

    return {
      success: true,
      data: { user },
    };
  } catch (error) {
    console.error("Error fetching current user:", error);
    return {
      success: false,
      error: "Failed to fetch user information. Please try again.",
    };
  }
}

/**
 * Sign out the current user and redirect to home page
 */
export async function signOutUser(): Promise<void> {
  try {
    await signOut({ redirectTo: "/" });
  } catch (error) {
    console.error("Error signing out user:", error);
    // Even if there's an error, redirect to home page
    redirect("/");
  }
}

/**
 * Validate user authentication status
 */
export async function validateAuth(): Promise<AuthActionResult> {
  try {
    const session = await auth();

    if (!session?.user?.id) {
      return {
        success: false,
        error: "Not authenticated",
        data: { isAuthenticated: false },
      };
    }

    // Verify user still exists in database
    const user = await getUserById(session.user.id);
    if (!user) {
      return {
        success: false,
        error: "User account not found",
        data: { isAuthenticated: false },
      };
    }

    return {
      success: true,
      data: {
        isAuthenticated: true,
        user,
        sessionExpires: session.expires,
      },
    };
  } catch (error) {
    console.error("Error validating authentication:", error);
    return {
      success: false,
      error: "Failed to validate authentication. Please try again.",
      data: { isAuthenticated: false },
    };
  }
}

/**
 * Refresh the current page data (useful after operations that change user state)
 */
export async function refreshUserData(
  path?: string
): Promise<AuthActionResult> {
  try {
    const session = await auth();
    if (!session?.user?.id) {
      return {
        success: false,
        error: "Not authenticated",
      };
    }

    // Revalidate the specified path or current dashboard paths
    const pathsToRevalidate = path
      ? [path]
      : ["/dashboard", "/dashboard/tokens", "/dashboard/sessions"];

    pathsToRevalidate.forEach((p) => {
      revalidatePath(p);
    });

    return {
      success: true,
      data: { message: "Data refreshed successfully" },
    };
  } catch (error) {
    console.error("Error refreshing user data:", error);
    return {
      success: false,
      error: "Failed to refresh data. Please try again.",
    };
  }
}
