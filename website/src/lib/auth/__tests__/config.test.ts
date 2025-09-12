import { describe, it, expect, beforeEach, vi } from "vitest";
import { redirect } from "next/navigation";
import {
  getRequiredSession,
  getOptionalSession,
  getRequiredUser,
  isAuthenticated,
  requireAuth,
} from "../config";
import { auth } from "@/auth";
import type { Session } from "next-auth";

// Mock dependencies
vi.mock("@/auth", () => ({
  auth: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  redirect: vi.fn(),
}));

const mockAuth = auth as vi.MockedFunction<typeof auth>;
const mockRedirect = redirect as vi.MockedFunction<typeof redirect>;

describe("Auth Config Utilities", () => {
  const mockSession: Session = {
    user: {
      id: "user-123",
      email: "test@example.com",
      name: "Test User",
      image: "https://example.com/avatar.jpg",
    },
    expires: "2024-12-31T23:59:59.999Z",
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("getRequiredSession", () => {
    it("should return session when user is authenticated", async () => {
      mockAuth.mockResolvedValue(mockSession);

      const result = await getRequiredSession();

      expect(result).toEqual(mockSession);
      expect(mockRedirect).not.toHaveBeenCalled();
    });

    it("should redirect to auth when no session", async () => {
      mockAuth.mockResolvedValue(null);

      await getRequiredSession();

      expect(mockRedirect).toHaveBeenCalledWith("/auth");
    });
  });

  describe("getOptionalSession", () => {
    it("should return session when user is authenticated", async () => {
      mockAuth.mockResolvedValue(mockSession);

      const result = await getOptionalSession();

      expect(result).toEqual(mockSession);
    });

    it("should return null when no session", async () => {
      mockAuth.mockResolvedValue(null);

      const result = await getOptionalSession();

      expect(result).toBeNull();
      expect(mockRedirect).not.toHaveBeenCalled();
    });
  });

  describe("getRequiredUser", () => {
    it("should return user when session exists with user", async () => {
      mockAuth.mockResolvedValue(mockSession);

      const result = await getRequiredUser();

      expect(result).toEqual(mockSession.user);
      expect(mockRedirect).not.toHaveBeenCalled();
    });

    it("should redirect when no session", async () => {
      mockAuth.mockResolvedValue(null);

      await getRequiredUser();

      expect(mockRedirect).toHaveBeenCalledWith("/auth");
    });

    it("should redirect when session exists but no user", async () => {
      const sessionWithoutUser = { ...mockSession, user: undefined };
      mockAuth.mockResolvedValue(sessionWithoutUser);

      await getRequiredUser();

      expect(mockRedirect).toHaveBeenCalledWith("/auth");
    });
  });

  describe("isAuthenticated", () => {
    it("should return true when user is authenticated", async () => {
      mockAuth.mockResolvedValue(mockSession);

      const result = await isAuthenticated();

      expect(result).toBe(true);
    });

    it("should return false when no session", async () => {
      mockAuth.mockResolvedValue(null);

      const result = await isAuthenticated();

      expect(result).toBe(false);
    });

    it("should return false when session exists but no user", async () => {
      const sessionWithoutUser = { ...mockSession, user: undefined };
      mockAuth.mockResolvedValue(sessionWithoutUser);

      const result = await isAuthenticated();

      expect(result).toBe(false);
    });
  });

  describe("requireAuth", () => {
    it("should not redirect when user is authenticated", async () => {
      mockAuth.mockResolvedValue(mockSession);

      await requireAuth();

      expect(mockRedirect).not.toHaveBeenCalled();
    });

    it("should redirect to auth when not authenticated", async () => {
      mockAuth.mockResolvedValue(null);

      await requireAuth();

      expect(mockRedirect).toHaveBeenCalledWith("/auth");
    });
  });
});
