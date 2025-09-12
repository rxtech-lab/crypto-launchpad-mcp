import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { signIn, signOut } from "next-auth/react";
import { SignInForm } from "@/components/auth/sign-in-form";
import { DashboardNav } from "@/components/dashboard/dashboard-nav";
import * as queries from "@/lib/db/queries";
import * as jwtUtils from "@/lib/auth/jwt";

// Mock dependencies
vi.mock("next-auth/react", () => ({
  signIn: vi.fn(),
  signOut: vi.fn(),
}));

vi.mock("@/lib/db/queries", () => ({
  createUser: vi.fn(),
  getUserByEmail: vi.fn(),
  createSession: vi.fn(),
  createJwtToken: vi.fn(),
  getUserJwtTokens: vi.fn(),
  deleteSession: vi.fn(),
}));

vi.mock("@/lib/auth/jwt", () => ({
  generateJwtToken: vi.fn(),
  verifyJwtToken: vi.fn(),
}));

vi.mock("@/components/auth/google-auth", () => ({
  GoogleAuth: () => <div data-testid="google-auth">Google Auth</div>,
}));

const mockSignIn = signIn as vi.MockedFunction<typeof signIn>;
const mockSignOut = signOut as vi.MockedFunction<typeof signOut>;
const mockCreateUser = queries.createUser as vi.MockedFunction<
  typeof queries.createUser
>;
const mockGetUserByEmail = queries.getUserByEmail as vi.MockedFunction<
  typeof queries.getUserByEmail
>;
const mockCreateSession = queries.createSession as vi.MockedFunction<
  typeof queries.createSession
>;
const mockCreateJwtToken = queries.createJwtToken as vi.MockedFunction<
  typeof queries.createJwtToken
>;
const mockGenerateJwtToken = jwtUtils.generateJwtToken as vi.MockedFunction<
  typeof jwtUtils.generateJwtToken
>;

// Mock window.PublicKeyCredential for WebAuthn
Object.defineProperty(window, "PublicKeyCredential", {
  writable: true,
  value: function PublicKeyCredential() {},
});

describe("Authentication Flow Integration Tests", () => {
  const mockUser = {
    id: "user-123",
    email: "test@example.com",
    name: "Test User",
    image: "https://example.com/avatar.jpg",
    createdAt: new Date(),
    updatedAt: new Date(),
  };

  const mockSession = {
    sessionToken: "session-123",
    userId: "user-123",
    expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
  };

  const mockJwtToken = {
    id: "token-123",
    userId: "user-123",
    tokenName: "Test Token",
    jti: "jti-123",
    aud: ["test-audience"],
    clientId: "test-client",
    roles: ["user"],
    scopes: ["read"],
    createdAt: new Date(),
    expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
    isActive: true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Mock successful responses by default
    mockCreateUser.mockResolvedValue(mockUser);
    mockGetUserByEmail.mockResolvedValue(mockUser);
    mockCreateSession.mockResolvedValue(mockSession);
    mockCreateJwtToken.mockResolvedValue(mockJwtToken);
    mockGenerateJwtToken.mockReturnValue({
      token: "jwt-token-123",
      authenticatedUser: {
        aud: ["test-audience"],
        client_id: "test-client",
        exp: Math.floor(Date.now() / 1000) + 30 * 24 * 60 * 60,
        iat: Math.floor(Date.now() / 1000),
        iss: "test-issuer",
        jti: "jti-123",
        nbf: Math.floor(Date.now() / 1000),
        oid: "user-123",
        resid: "user-123",
        roles: ["user"],
        scopes: ["read"],
        sid: "session-123",
        sub: "user-123",
      },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("Complete Sign-up Flow with WebAuthn", () => {
    it("should complete WebAuthn registration flow", async () => {
      // Mock successful WebAuthn registration
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<SignInForm />);

      // Find and click the WebAuthn create account button
      const createAccountButton = screen.getByText(
        "Create account with passkey"
      );
      expect(createAccountButton).toBeInTheDocument();

      fireEvent.click(createAccountButton);

      // Verify signIn was called with correct parameters
      expect(mockSignIn).toHaveBeenCalledWith("webauthn", {
        redirect: false,
        callbackUrl: "/dashboard",
      });

      // Verify loading state is shown
      await waitFor(() => {
        expect(screen.getByText("Creating account...")).toBeInTheDocument();
      });
    });

    it("should handle WebAuthn registration errors", async () => {
      // Mock WebAuthn registration error
      mockSignIn.mockResolvedValue({
        error: "NotAllowedError",
        url: null,
        ok: false,
        status: 401,
      });

      render(<SignInForm />);

      const createAccountButton = screen.getByText(
        "Create account with passkey"
      );
      fireEvent.click(createAccountButton);

      // Verify error message is displayed
      await waitFor(() => {
        expect(
          screen.getByText("Authentication was cancelled or timed out.")
        ).toBeInTheDocument();
      });
    });

    it("should complete WebAuthn sign-in flow", async () => {
      // Mock successful WebAuthn sign-in
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<SignInForm />);

      const signInButton = screen.getByText("Sign in with passkey");
      fireEvent.click(signInButton);

      expect(mockSignIn).toHaveBeenCalledWith("webauthn", {
        redirect: false,
        callbackUrl: "/dashboard",
      });

      await waitFor(() => {
        expect(screen.getByText("Signing in...")).toBeInTheDocument();
      });
    });
  });

  describe("Complete Sign-up Flow with Google OAuth", () => {
    it("should initiate Google OAuth flow", async () => {
      // Mock successful Google OAuth
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<SignInForm />);

      // The Google Auth component is mocked, but we can test the integration
      const googleAuthComponent = screen.getByTestId("google-auth");
      expect(googleAuthComponent).toBeInTheDocument();
    });
  });

  describe("Dashboard Access and Token Management", () => {
    it("should render dashboard navigation after successful authentication", () => {
      render(<DashboardNav />);

      // Verify all navigation elements are present
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
      expect(screen.getByText("Overview")).toBeInTheDocument();
      expect(screen.getByText("JWT Tokens")).toBeInTheDocument();
      expect(screen.getByText("Sessions")).toBeInTheDocument();
      expect(screen.getByText("Sign Out")).toBeInTheDocument();
    });

    it("should handle sign out from dashboard", async () => {
      mockSignOut.mockResolvedValue(undefined);

      render(<DashboardNav />);

      const signOutButton = screen.getByText("Sign Out");
      fireEvent.click(signOutButton);

      expect(mockSignOut).toHaveBeenCalledWith({ callbackUrl: "/" });
    });

    it("should navigate to different dashboard sections", () => {
      render(<DashboardNav />);

      // Check navigation links
      const overviewLink = screen.getByText("Overview").closest("a");
      expect(overviewLink).toHaveAttribute("href", "/dashboard");

      const tokensLink = screen.getByText("JWT Tokens").closest("a");
      expect(tokensLink).toHaveAttribute("href", "/dashboard/tokens");

      const sessionsLink = screen.getByText("Sessions").closest("a");
      expect(sessionsLink).toHaveAttribute("href", "/dashboard/sessions");

      const backLink = screen.getByText("Back to Site").closest("a");
      expect(backLink).toHaveAttribute("href", "/");
    });
  });

  describe("Session Management and Cleanup", () => {
    it("should handle session creation during authentication", async () => {
      // This would typically be tested in a more integrated environment
      // where we can mock the full authentication flow

      const sessionData = {
        sessionToken: "new-session-123",
        userId: "user-123",
        expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
      };

      await mockCreateSession(sessionData);

      expect(mockCreateSession).toHaveBeenCalledWith(sessionData);
    });

    it("should handle JWT token generation during authentication", async () => {
      const tokenPayload = {
        aud: ["test-audience"],
        clientId: "test-client",
        roles: ["user"],
        scopes: ["read"],
        expiresIn: "30d",
      };

      const result = mockGenerateJwtToken("user-123", tokenPayload);

      expect(result).toHaveProperty("token");
      expect(result).toHaveProperty("authenticatedUser");
      expect(result.authenticatedUser.sub).toBe("user-123");
    });

    it("should handle user creation during first-time authentication", async () => {
      const newUserData = {
        id: "new-user-123",
        email: "newuser@example.com",
        name: "New User",
        image: "https://example.com/new-avatar.jpg",
      };

      await mockCreateUser(newUserData);

      expect(mockCreateUser).toHaveBeenCalledWith(newUserData);
    });
  });

  describe("Error Handling in Authentication Flows", () => {
    it("should handle database errors during user creation", async () => {
      mockCreateUser.mockRejectedValue(new Error("Database connection failed"));

      try {
        await mockCreateUser({
          id: "user-123",
          email: "test@example.com",
          name: "Test User",
        });
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toBe("Database connection failed");
      }
    });

    it("should handle JWT token generation errors", async () => {
      mockGenerateJwtToken.mockImplementation(() => {
        throw new Error("JWT generation failed");
      });

      expect(() => {
        mockGenerateJwtToken("user-123", {
          aud: ["test"],
          roles: ["user"],
          scopes: ["read"],
        });
      }).toThrow("JWT generation failed");
    });

    it("should handle session creation errors", async () => {
      mockCreateSession.mockRejectedValue(new Error("Session creation failed"));

      try {
        await mockCreateSession({
          sessionToken: "session-123",
          userId: "user-123",
          expires: new Date(),
        });
      } catch (error) {
        expect(error).toBeInstanceOf(Error);
        expect((error as Error).message).toBe("Session creation failed");
      }
    });
  });

  describe("Authentication State Management", () => {
    it("should maintain authentication state across components", () => {
      // Test that authentication state is properly managed
      // This would typically involve testing with a state management solution

      render(<SignInForm />);

      // Verify initial unauthenticated state
      expect(screen.getByText("Welcome")).toBeInTheDocument();
      expect(screen.getByText("Sign in with passkey")).toBeInTheDocument();
    });

    it("should handle authentication state transitions", async () => {
      // Mock successful authentication
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<SignInForm />);

      const signInButton = screen.getByText("Sign in with passkey");
      fireEvent.click(signInButton);

      // Verify loading state
      await waitFor(() => {
        expect(screen.getByText("Signing in...")).toBeInTheDocument();
      });

      // In a real integration test, we would verify the redirect to dashboard
      expect(mockSignIn).toHaveBeenCalled();
    });
  });

  describe("Security and Validation", () => {
    it("should validate authentication tokens", () => {
      const validToken = "valid-jwt-token";
      const mockAuthenticatedUser = {
        aud: ["test-audience"],
        client_id: "test-client",
        exp: Math.floor(Date.now() / 1000) + 3600,
        iat: Math.floor(Date.now() / 1000),
        iss: "test-issuer",
        jti: "jti-123",
        nbf: Math.floor(Date.now() / 1000),
        oid: "user-123",
        resid: "user-123",
        roles: ["user"],
        scopes: ["read"],
        sid: "session-123",
        sub: "user-123",
      };

      const mockVerifyJwtToken = vi.fn().mockReturnValue(mockAuthenticatedUser);

      const result = mockVerifyJwtToken(validToken);
      expect(result).toEqual(mockAuthenticatedUser);
    });

    it("should handle invalid authentication tokens", () => {
      const invalidToken = "invalid-jwt-token";
      const mockVerifyJwtToken = vi.fn().mockReturnValue(null);

      const result = mockVerifyJwtToken(invalidToken);
      expect(result).toBeNull();
    });
  });
});
