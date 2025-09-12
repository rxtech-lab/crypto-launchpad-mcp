import { eq, and, desc } from "drizzle-orm";
import { db } from "./connection";
import {
  users,
  sessions,
  jwtTokens,
  authenticators,
  type User,
  type NewUser,
  type Session,
  type NewSession,
  type JwtToken,
  type NewJwtToken,
  type Authenticator,
  type NewAuthenticator,
} from "./schema";

// User queries
export async function createUser(userData: NewUser): Promise<User> {
  try {
    const [user] = await db.insert(users).values(userData).returning();
    return user;
  } catch (error) {
    console.error("Error creating user:", error);
    throw new Error("Failed to create user");
  }
}

export async function getUserById(id: string): Promise<User | null> {
  try {
    const [user] = await db.select().from(users).where(eq(users.id, id));
    return user || null;
  } catch (error) {
    console.error("Error fetching user by ID:", error);
    throw new Error("Failed to fetch user");
  }
}

export async function getUserByEmail(email: string): Promise<User | null> {
  try {
    const [user] = await db.select().from(users).where(eq(users.email, email));
    return user || null;
  } catch (error) {
    console.error("Error fetching user by email:", error);
    throw new Error("Failed to fetch user");
  }
}

// Session queries
export async function createSession(sessionData: NewSession): Promise<Session> {
  try {
    const [session] = await db.insert(sessions).values(sessionData).returning();
    return session;
  } catch (error) {
    console.error("Error creating session:", error);
    throw new Error("Failed to create session");
  }
}

export async function getSessionByToken(
  sessionToken: string
): Promise<Session | null> {
  try {
    const [session] = await db
      .select()
      .from(sessions)
      .where(eq(sessions.sessionToken, sessionToken));
    return session || null;
  } catch (error) {
    console.error("Error fetching session by token:", error);
    throw new Error("Failed to fetch session");
  }
}

export async function getUserSessions(userId: string): Promise<Session[]> {
  try {
    return await db
      .select()
      .from(sessions)
      .where(eq(sessions.userId, userId))
      .orderBy(desc(sessions.expires));
  } catch (error) {
    console.error("Error fetching user sessions:", error);
    throw new Error("Failed to fetch user sessions");
  }
}

export async function deleteSession(sessionToken: string): Promise<void> {
  try {
    await db.delete(sessions).where(eq(sessions.sessionToken, sessionToken));
  } catch (error) {
    console.error("Error deleting session:", error);
    throw new Error("Failed to delete session");
  }
}

// JWT Token queries
export async function createJwtToken(
  tokenData: NewJwtToken
): Promise<JwtToken> {
  try {
    const [token] = await db.insert(jwtTokens).values(tokenData).returning();
    return token;
  } catch (error) {
    console.error("Error creating JWT token:", error);
    throw new Error("Failed to create JWT token");
  }
}

export async function getUserJwtTokens(userId: string): Promise<JwtToken[]> {
  try {
    return await db
      .select()
      .from(jwtTokens)
      .where(and(eq(jwtTokens.userId, userId), eq(jwtTokens.isActive, true)))
      .orderBy(desc(jwtTokens.createdAt));
  } catch (error) {
    console.error("Error fetching user JWT tokens:", error);
    throw new Error("Failed to fetch JWT tokens");
  }
}

export async function getJwtTokenByJti(jti: string): Promise<JwtToken | null> {
  try {
    const [token] = await db
      .select()
      .from(jwtTokens)
      .where(eq(jwtTokens.jti, jti));
    return token || null;
  } catch (error) {
    console.error("Error fetching JWT token by JTI:", error);
    throw new Error("Failed to fetch JWT token");
  }
}

export async function deactivateJwtToken(id: string): Promise<void> {
  try {
    await db
      .update(jwtTokens)
      .set({
        isActive: false,
      })
      .where(eq(jwtTokens.id, id));
  } catch (error) {
    console.error("Error deactivating JWT token:", error);
    throw new Error("Failed to deactivate JWT token");
  }
}

export async function deleteJwtToken(id: string): Promise<void> {
  try {
    await db.delete(jwtTokens).where(eq(jwtTokens.id, id));
  } catch (error) {
    console.error("Error deleting JWT token:", error);
    throw new Error("Failed to delete JWT token");
  }
}

// WebAuthn Authenticator queries
export async function createAuthenticator(
  authenticatorData: NewAuthenticator
): Promise<Authenticator> {
  try {
    const [authenticator] = await db
      .insert(authenticators)
      .values(authenticatorData)
      .returning();
    return authenticator;
  } catch (error) {
    console.error("Error creating authenticator:", error);
    throw new Error("Failed to create authenticator");
  }
}

export async function getUserAuthenticators(
  userId: string
): Promise<Authenticator[]> {
  try {
    return await db
      .select()
      .from(authenticators)
      .where(eq(authenticators.userId, userId));
  } catch (error) {
    console.error("Error fetching user authenticators:", error);
    throw new Error("Failed to fetch authenticators");
  }
}

export async function getAuthenticatorByCredentialId(
  credentialId: string
): Promise<Authenticator | null> {
  try {
    const [authenticator] = await db
      .select()
      .from(authenticators)
      .where(eq(authenticators.credentialID, credentialId));
    return authenticator || null;
  } catch (error) {
    console.error("Error fetching authenticator by credential ID:", error);
    throw new Error("Failed to fetch authenticator");
  }
}

export async function updateAuthenticatorCounter(
  credentialId: string,
  counter: number
): Promise<void> {
  try {
    await db
      .update(authenticators)
      .set({ counter })
      .where(eq(authenticators.credentialID, credentialId));
  } catch (error) {
    console.error("Error updating authenticator counter:", error);
    throw new Error("Failed to update authenticator counter");
  }
}

export async function deleteAuthenticator(credentialId: string): Promise<void> {
  try {
    await db
      .delete(authenticators)
      .where(eq(authenticators.credentialID, credentialId));
  } catch (error) {
    console.error("Error deleting authenticator:", error);
    throw new Error("Failed to delete authenticator");
  }
}
