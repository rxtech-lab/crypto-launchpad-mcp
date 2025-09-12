// Validation-related type definitions

// Validation result type
export interface ValidationResult<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  errors?: ValidationError[];
}

// Individual validation error
export interface ValidationError {
  field: string;
  message: string;
  code?: string;
}

// Form validation states
export interface FormValidationState {
  isValid: boolean;
  isValidating: boolean;
  errors: Record<string, string>;
  touched: Record<string, boolean>;
}

// Input validation rules
export interface ValidationRule {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  pattern?: RegExp;
  custom?: (value: any) => boolean | string;
}

// Field validation configuration
export interface FieldValidation {
  [fieldName: string]: ValidationRule;
}

// Token creation form validation
export interface TokenFormValidation extends FormValidationState {
  fields: {
    tokenName: string;
    clientId: string;
    expiresIn: string;
    roles: string[];
    scopes: string[];
    aud: string[];
  };
}

// Session validation types
export interface SessionValidation {
  isValidSession: boolean;
  isExpired: boolean;
  belongsToUser: boolean;
  isCurrentSession: boolean;
  error?: string;
}

// Authentication validation
export interface AuthValidation {
  isAuthenticated: boolean;
  hasValidSession: boolean;
  userExists: boolean;
  error?: string;
}

// API input validation schemas (for runtime validation)
export interface ApiValidationSchema {
  body?: Record<string, ValidationRule>;
  query?: Record<string, ValidationRule>;
  params?: Record<string, ValidationRule>;
}

// Validation middleware result
export interface ValidationMiddlewareResult {
  isValid: boolean;
  validatedData?: any;
  errors?: ValidationError[];
}
