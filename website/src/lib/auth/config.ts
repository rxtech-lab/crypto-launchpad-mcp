import { auth } from "@/auth";
import { redirect } from "next/navigation";
import type { Session } from "next-auth";

/**
 * Server-side function to get the current session
 * Throws an error if no session is found
 */
export async function getRequiredSession(): Promise<Session> {
  const session = await auth();

  if (!session) {
    redirect("/auth");
  }

  return session;
}

/**
 * Server-side function to get the current session
 * Returns null if no session is found
 */
export async function getOptionalSession(): Promise<Session | null> {
  return await auth();
}

/**
 * Server-side function to get the current user
 * Throws an error if no user is found
 */
export async function getRequiredUser() {
  const session = await getRequiredSession();

  if (!session.user) {
    redirect("/auth");
  }

  return session.user;
}

/**
 * Server-side function to check if user is authenticated
 */
export async function isAuthenticated(): Promise<boolean> {
  const session = await auth();
  return !!session?.user;
}

/**
 * Redirect to auth page if not authenticated
 */
export async function requireAuth() {
  const authenticated = await isAuthenticated();

  if (!authenticated) {
    redirect("/auth");
  }
}
