import { describe, it, expect, beforeEach, vi } from "vitest";
import jwt from "jsonwebtoken";
import {
  generateJwtToken,
  verifyJwtToken,
  getExpirationTime,
  getExpirationDate,
} from "../jwt";
import type { JwtTokenPayload } from "@/types/auth";

// Mock uuid
vi.mock("uuid", () => ({
  v4: vi.fn(() => "mock-uuid-123"),
}));

describe("JWT Utilities", () => {
  const mockUserId = "user-123";
  const mockPayload: JwtTokenPayload = {
    aud: ["test-audience"],
    clientId: "test-client",
    roles: ["user"],
    scopes: ["read", "write"],
    expiresIn: "7d",
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset Date.now mock
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-01T00:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("generateJwtToken", () => {
    it("should generate a valid JWT token with correct structure", () => {
      const result = generateJwtToken(mockUserId, mockPayload);

      expect(result).toHaveProperty("token");
      expect(result).toHaveProperty("authenticatedUser");
      expect(typeof result.token).toBe("string");

      const { authenticatedUser } = result;
      expect(authenticatedUser.aud).toEqual(mockPayload.aud);
      expect(authenticatedUser.client_id).toBe(mockPayload.clientId);
      expect(authenticatedUser.roles).toEqual(mockPayload.roles);
      expect(authenticatedUser.scopes).toEqual(mockPayload.scopes);
      expect(authenticatedUser.sub).toBe(mockUserId);
      expect(authenticatedUser.oid).toBe(mockUserId);
      expect(authenticatedUser.resid).toBe(mockUserId);
      expect(authenticatedUser.iss).toBe("test-issuer");
      expect(authenticatedUser.jti).toBe("mock-uuid-123");
    });

    it("should set correct timestamps", () => {
      const result = generateJwtToken(mockUserId, mockPayload);
      const { authenticatedUser } = result;

      const expectedIat = Math.floor(Date.now() / 1000);
      expect(authenticatedUser.iat).toBe(expectedIat);
      expect(authenticatedUser.nbf).toBe(expectedIat);
      expect(authenticatedUser.exp).toBeGreaterThan(expectedIat);
    });

    it("should use default client_id when not provided", () => {
      const payloadWithoutClientId = { ...mockPayload };
      delete payloadWithoutClientId.clientId;

      const result = generateJwtToken(mockUserId, payloadWithoutClientId);
      expect(result.authenticatedUser.client_id).toBe("client-mock-uuid-123");
    });

    it("should use default expiration when not provided", () => {
      const payloadWithoutExpiration = { ...mockPayload };
      delete payloadWithoutExpiration.expiresIn;

      const result = generateJwtToken(mockUserId, payloadWithoutExpiration);
      const expectedExp = Math.floor(Date.now() / 1000) + 30 * 24 * 60 * 60; // 30 days
      expect(result.authenticatedUser.exp).toBe(expectedExp);
    });
  });

  describe("verifyJwtToken", () => {
    it("should verify a valid JWT token", () => {
      const { token, authenticatedUser } = generateJwtToken(
        mockUserId,
        mockPayload
      );
      const verified = verifyJwtToken(token);

      expect(verified).toEqual(authenticatedUser);
    });

    it("should return null for invalid token", () => {
      const invalidToken = "invalid.jwt.token";
      const verified = verifyJwtToken(invalidToken);

      expect(verified).toBeNull();
    });

    it("should return null for expired token", () => {
      // Create a token that's already expired
      const expiredPayload = { ...mockPayload, expiresIn: "0d" };
      const { token } = generateJwtToken(mockUserId, expiredPayload);

      // Move time forward
      vi.setSystemTime(new Date("2024-01-02T00:00:00Z"));

      const verified = verifyJwtToken(token);
      expect(verified).toBeNull();
    });

    it("should return null for token with wrong secret", () => {
      // Create token with different secret
      const wrongSecretToken = jwt.sign({ sub: mockUserId }, "wrong-secret");
      const verified = verifyJwtToken(wrongSecretToken);

      expect(verified).toBeNull();
    });
  });

  describe("getExpirationTime", () => {
    it("should calculate correct expiration for days", () => {
      const now = Math.floor(Date.now() / 1000);
      const expiration = getExpirationTime("7d");
      const expected = now + 7 * 24 * 60 * 60;

      expect(expiration).toBe(expected);
    });

    it("should calculate correct expiration for weeks", () => {
      const now = Math.floor(Date.now() / 1000);
      const expiration = getExpirationTime("2w");
      const expected = now + 2 * 7 * 24 * 60 * 60;

      expect(expiration).toBe(expected);
    });

    it("should calculate correct expiration for months", () => {
      const now = Math.floor(Date.now() / 1000);
      const expiration = getExpirationTime("3m");
      const expected = now + 3 * 30 * 24 * 60 * 60;

      expect(expiration).toBe(expected);
    });

    it("should calculate correct expiration for years", () => {
      const now = Math.floor(Date.now() / 1000);
      const expiration = getExpirationTime("1y");
      const expected = now + 1 * 365 * 24 * 60 * 60;

      expect(expiration).toBe(expected);
    });

    it("should throw error for invalid format", () => {
      expect(() => getExpirationTime("invalid")).toThrow(
        "Invalid expiration format. Use format like '7d', '30d', '1y'"
      );
    });

    it("should throw error for invalid unit", () => {
      expect(() => getExpirationTime("7x")).toThrow(
        "Invalid time unit. Use 'd', 'w', 'm', or 'y'"
      );
    });
  });

  describe("getExpirationDate", () => {
    it("should return correct Date object", () => {
      const expirationDate = getExpirationDate("1d");
      const expectedTimestamp = Math.floor(Date.now() / 1000) + 24 * 60 * 60;
      const expectedDate = new Date(expectedTimestamp * 1000);

      expect(expirationDate).toEqual(expectedDate);
    });
  });
});
