import { z } from "zod";

// Schema for creating a new JWT token
export const createTokenSchema = z.object({
  tokenName: z
    .string()
    .min(1, "Token name is required")
    .max(100, "Token name must be less than 100 characters")
    .trim(),
  aud: z.array(z.string()).optional().default([]),
  clientId: z
    .string()
    .optional()
    .transform((val) => val?.trim() || undefined),
  roles: z.array(z.string()).optional().default([]),
  scopes: z.array(z.string()).optional().default([]),
  expiresIn: z
    .string()
    .regex(
      /^\d+[dwmy]$/,
      "Invalid expiration format. Use format like '7d', '30d', '1y'"
    )
    .optional()
    .default("30d"),
});

// Schema for token ID validation
export const tokenIdSchema = z
  .string()
  .min(1, "Token ID is required")
  .uuid("Invalid token ID format");

// Schema for JTI validation
export const jtiSchema = z
  .string()
  .min(1, "JTI is required")
  .uuid("Invalid JTI format");

// Type exports
export type CreateTokenInput = z.infer<typeof createTokenSchema>;
export type TokenIdInput = z.infer<typeof tokenIdSchema>;
export type JtiInput = z.infer<typeof jtiSchema>;

// Validation functions
export function validateCreateToken(data: unknown): {
  success: boolean;
  data?: CreateTokenInput;
  error?: string;
} {
  try {
    const result = createTokenSchema.parse(data);
    return { success: true, data: result };
  } catch (error) {
    if (error instanceof z.ZodError) {
      const firstError = error.issues[0];
      return { success: false, error: firstError.message };
    }
    return { success: false, error: "Invalid input data" };
  }
}

export function validateTokenId(tokenId: unknown): {
  success: boolean;
  data?: string;
  error?: string;
} {
  try {
    const result = tokenIdSchema.parse(tokenId);
    return { success: true, data: result };
  } catch (error) {
    if (error instanceof z.ZodError) {
      const firstError = error.issues[0];
      return { success: false, error: firstError.message };
    }
    return { success: false, error: "Invalid token ID" };
  }
}

export function validateJti(jti: unknown): {
  success: boolean;
  data?: string;
  error?: string;
} {
  try {
    const result = jtiSchema.parse(jti);
    return { success: true, data: result };
  } catch (error) {
    if (error instanceof z.ZodError) {
      const firstError = error.issues[0];
      return { success: false, error: firstError.message };
    }
    return { success: false, error: "Invalid JTI" };
  }
}
