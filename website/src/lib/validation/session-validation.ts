import { z } from "zod";

// Schema for session token validation
export const sessionTokenSchema = z
  .string()
  .min(1, "Session token is required")
  .max(500, "Session token is too long");

// Type exports
export type SessionTokenInput = z.infer<typeof sessionTokenSchema>;

// Validation functions
export function validateSessionToken(sessionToken: unknown): {
  success: boolean;
  data?: string;
  error?: string;
} {
  try {
    const result = sessionTokenSchema.parse(sessionToken);
    return { success: true, data: result };
  } catch (error) {
    if (error instanceof z.ZodError) {
      const firstError = error.issues[0];
      return { success: false, error: firstError.message };
    }
    return { success: false, error: "Invalid session token" };
  }
}

// Additional validation for session operations
export function validateSessionOperation(
  sessionToken: string,
  currentSessionToken?: string
): {
  success: boolean;
  error?: string;
} {
  // Prevent users from deleting their current session
  if (sessionToken === currentSessionToken) {
    return {
      success: false,
      error: "Cannot perform this operation on your current session",
    };
  }

  return { success: true };
}
