/**
 * Standardized error handling for server actions
 */

export interface ActionError {
  code: string;
  message: string;
  details?: any;
}

export class ActionErrorHandler {
  static handleDatabaseError(error: any): ActionError {
    console.error("Database error:", error);

    // Handle specific database errors
    if (error.code === "23505") {
      // Unique constraint violation
      return {
        code: "DUPLICATE_ENTRY",
        message: "A record with this information already exists",
        details: error.detail,
      };
    }

    if (error.code === "23503") {
      // Foreign key constraint violation
      return {
        code: "INVALID_REFERENCE",
        message: "Referenced record does not exist",
        details: error.detail,
      };
    }

    if (error.code === "23502") {
      // Not null constraint violation
      return {
        code: "MISSING_REQUIRED_FIELD",
        message: "Required field is missing",
        details: error.detail,
      };
    }

    return {
      code: "DATABASE_ERROR",
      message: "A database error occurred. Please try again.",
    };
  }

  static handleAuthError(error: any): ActionError {
    console.error("Authentication error:", error);

    if (error.type === "AccessDenied") {
      return {
        code: "ACCESS_DENIED",
        message: "You don't have permission to perform this action",
      };
    }

    if (error.type === "SessionRequired") {
      return {
        code: "SESSION_REQUIRED",
        message: "You must be signed in to perform this action",
      };
    }

    return {
      code: "AUTH_ERROR",
      message: "An authentication error occurred. Please try again.",
    };
  }

  static handleValidationError(error: any): ActionError {
    console.error("Validation error:", error);

    return {
      code: "VALIDATION_ERROR",
      message: error.message || "Invalid input data",
      details: error.errors,
    };
  }

  static handleJWTError(error: any): ActionError {
    console.error("JWT error:", error);

    if (error.name === "TokenExpiredError") {
      return {
        code: "TOKEN_EXPIRED",
        message: "Token has expired",
      };
    }

    if (error.name === "JsonWebTokenError") {
      return {
        code: "INVALID_TOKEN",
        message: "Invalid token format",
      };
    }

    return {
      code: "JWT_ERROR",
      message: "Token processing error. Please try again.",
    };
  }

  static handleGenericError(error: any): ActionError {
    console.error("Generic error:", error);

    return {
      code: "INTERNAL_ERROR",
      message: "An unexpected error occurred. Please try again.",
    };
  }

  /**
   * Main error handler that routes to specific handlers based on error type
   */
  static handle(error: any): ActionError {
    if (error.name === "ZodError") {
      return this.handleValidationError(error);
    }

    if (error.name?.includes("JWT") || error.name?.includes("Token")) {
      return this.handleJWTError(error);
    }

    if (error.code && typeof error.code === "string") {
      return this.handleDatabaseError(error);
    }

    if (
      error.type &&
      (error.type === "AccessDenied" || error.type === "SessionRequired")
    ) {
      return this.handleAuthError(error);
    }

    return this.handleGenericError(error);
  }
}

/**
 * Utility function to create standardized error responses
 */
export function createErrorResponse(error: any) {
  const actionError = ActionErrorHandler.handle(error);

  return {
    success: false,
    error: actionError.message,
    errorCode: actionError.code,
    errorDetails: actionError.details,
  };
}

/**
 * Utility function to create success responses
 */
export function createSuccessResponse(data?: any, message?: string) {
  return {
    success: true,
    data,
    message,
  };
}
