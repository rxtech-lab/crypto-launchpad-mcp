import jwt from "jsonwebtoken";
import { v4 as uuidv4 } from "uuid";
import type { AuthenticatedUser, JwtTokenPayload } from "@/types/auth";

const JWT_SECRET = process.env.JWT_SECRET || "your-jwt-secret";
const JWT_ISSUER = process.env.JWT_ISSUER || "crypto-launchpad";

export function generateJwtToken(
  userId: string,
  payload: JwtTokenPayload
): { token: string; authenticatedUser: AuthenticatedUser } {
  const now = Math.floor(Date.now() / 1000);
  const jti = uuidv4();
  const sessionId = uuidv4();

  // Calculate expiration (default to 30 days if not specified)
  const expiresIn = payload.expiresIn || "30d";
  const expirationTime = getExpirationTime(expiresIn);

  const authenticatedUser: AuthenticatedUser = {
    aud: payload.aud,
    client_id: payload.clientId || `client-${uuidv4()}`,
    exp: expirationTime,
    iat: now,
    iss: JWT_ISSUER,
    jti,
    nbf: now,
    oid: userId, // Object ID (user ID)
    resid: userId, // Resource ID (user ID)
    roles: payload.roles,
    scopes: payload.scopes,
    sid: sessionId, // Session ID
    sub: userId, // Subject (user ID)
  };

  const token = jwt.sign(authenticatedUser, JWT_SECRET, {
    algorithm: "HS256",
  });

  return { token, authenticatedUser };
}

export function verifyJwtToken(token: string): AuthenticatedUser | null {
  try {
    const decoded = jwt.verify(token, JWT_SECRET) as AuthenticatedUser;
    return decoded;
  } catch (error) {
    console.error("JWT verification failed:", error);
    return null;
  }
}

export function getExpirationTime(expiresIn: string): number {
  const now = Math.floor(Date.now() / 1000);

  // Parse expiration string (e.g., "7d", "30d", "1y")
  const match = expiresIn.match(/^(\d+)([dwmy])$/);
  if (!match) {
    throw new Error(
      "Invalid expiration format. Use format like '7d', '30d', '1y'"
    );
  }

  const [, amount, unit] = match;
  const numAmount = parseInt(amount, 10);

  switch (unit) {
    case "d": // days
      return now + numAmount * 24 * 60 * 60;
    case "w": // weeks
      return now + numAmount * 7 * 24 * 60 * 60;
    case "m": // months (30 days)
      return now + numAmount * 30 * 24 * 60 * 60;
    case "y": // years (365 days)
      return now + numAmount * 365 * 24 * 60 * 60;
    default:
      throw new Error("Invalid time unit. Use 'd', 'w', 'm', or 'y'");
  }
}

export function getExpirationDate(expiresIn: string): Date {
  const timestamp = getExpirationTime(expiresIn);
  return new Date(timestamp * 1000);
}
