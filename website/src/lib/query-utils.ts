import { UseQueryResult, UseMutationResult } from "@tanstack/react-query";
import { ApiError } from "./query-client";

// Type guards for error handling
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function isNetworkError(error: unknown): error is TypeError {
  return error instanceof TypeError && error.message.includes("fetch");
}

// Error message extraction utilities
export function getErrorMessage(error: unknown): string {
  if (isApiError(error)) {
    return error.message;
  }

  if (isNetworkError(error)) {
    return "Network error. Please check your connection and try again.";
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "An unexpected error occurred. Please try again.";
}

export function getErrorStatus(error: unknown): number | null {
  if (isApiError(error)) {
    return error.status;
  }

  return null;
}

// Query state utilities
export function isQueryLoading(query: UseQueryResult): boolean {
  return query.isLoading || query.isFetching;
}

export function isQueryError(query: UseQueryResult): boolean {
  return query.isError && !query.isFetching;
}

export function isMutationLoading(mutation: UseMutationResult): boolean {
  return mutation.isPending;
}

// Retry utilities
export function shouldRetryQuery(error: unknown): boolean {
  // Don't retry on client errors (4xx)
  if (isApiError(error) && error.status >= 400 && error.status < 500) {
    return false;
  }

  // Don't retry on network errors in development
  if (process.env.NODE_ENV === "development" && isNetworkError(error)) {
    return false;
  }

  return true;
}

// Cache invalidation helpers
export function getQueryKeyPrefix(
  type: "auth" | "tokens" | "sessions"
): string[] {
  return [type];
}

// Loading state helpers for UI
export interface LoadingState {
  isLoading: boolean;
  isError: boolean;
  error: string | null;
}

export function getLoadingState(query: UseQueryResult): LoadingState {
  return {
    isLoading: isQueryLoading(query),
    isError: isQueryError(query),
    error: query.error ? getErrorMessage(query.error) : null,
  };
}

export function getMutationState(mutation: UseMutationResult): LoadingState {
  return {
    isLoading: isMutationLoading(mutation),
    isError: mutation.isError,
    error: mutation.error ? getErrorMessage(mutation.error) : null,
  };
}

// Optimistic update helpers
export function createOptimisticUpdate<T>(
  currentData: T[] | undefined,
  newItem: T,
  getId: (item: T) => string
): T[] {
  if (!currentData) return [newItem];

  const existingIndex = currentData.findIndex(
    (item) => getId(item) === getId(newItem)
  );

  if (existingIndex >= 0) {
    // Update existing item
    const updated = [...currentData];
    updated[existingIndex] = newItem;
    return updated;
  } else {
    // Add new item
    return [...currentData, newItem];
  }
}

export function createOptimisticDelete<T>(
  currentData: T[] | undefined,
  itemId: string,
  getId: (item: T) => string
): T[] {
  if (!currentData) return [];

  return currentData.filter((item) => getId(item) !== itemId);
}
