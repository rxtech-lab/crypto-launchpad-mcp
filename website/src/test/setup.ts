import "@testing-library/jest-dom";
import { vi } from "vitest";

// Mock environment variables
process.env.JWT_SECRET = "test-jwt-secret";
process.env.JWT_ISSUER = "test-issuer";
process.env.DATABASE_URL = "postgresql://test:test@localhost:5432/test";

// Mock console methods to reduce noise in tests
global.console = {
  ...console,
  error: vi.fn(),
  warn: vi.fn(),
  log: vi.fn(),
};
