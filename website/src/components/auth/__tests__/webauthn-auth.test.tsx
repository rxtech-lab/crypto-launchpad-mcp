import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { signIn } from "next-auth/react";
import { WebAuthnAuth } from "../webauthn-auth";

// Mock next-auth/react
vi.mock("next-auth/react", () => ({
  signIn: vi.fn(),
}));

const mockSignIn = signIn as vi.MockedFunction<typeof signIn>;

// Mock window.PublicKeyCredential for WebAuthn support
Object.defineProperty(window, "PublicKeyCredential", {
  writable: true,
  value: function PublicKeyCredential() {},
});

describe("WebAuthnAuth", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("WebAuthn Support", () => {
    it("should render WebAuthn interface when supported", () => {
      render(<WebAuthnAuth />);

      expect(screen.getByText("Passkey Authentication")).toBeInTheDocument();
      expect(screen.getByText("Sign in with passkey")).toBeInTheDocument();
      expect(
        screen.getByText("Create account with passkey")
      ).toBeInTheDocument();
    });

    it("should show unsupported message when WebAuthn is not available", () => {
      // Temporarily remove WebAuthn support
      const originalPublicKeyCredential = window.PublicKeyCredential;
      // @ts-ignore
      delete window.PublicKeyCredential;

      render(<WebAuthnAuth />);

      expect(
        screen.getByText("WebAuthn is not supported on this device or browser.")
      ).toBeInTheDocument();

      // Restore WebAuthn support
      window.PublicKeyCredential = originalPublicKeyCredential;
    });
  });

  describe("Authentication Flow", () => {
    it("should handle sign in button click", async () => {
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<WebAuthnAuth />);

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

    it("should handle create account button click", async () => {
      mockSignIn.mockResolvedValue({
        error: null,
        url: "/dashboard",
        ok: true,
        status: 200,
      });

      render(<WebAuthnAuth />);

      const createAccountButton = screen.getByText(
        "Create account with passkey"
      );
      fireEvent.click(createAccountButton);

      expect(mockSignIn).toHaveBeenCalledWith("webauthn", {
        redirect: false,
        callbackUrl: "/dashboard",
      });

      await waitFor(() => {
        expect(screen.getByText("Creating account...")).toBeInTheDocument();
      });
    });

    it("should display error message when authentication fails", async () => {
      mockSignIn.mockResolvedValue({
        error: "NotAllowedError",
        url: null,
        ok: false,
        status: 401,
      });

      render(<WebAuthnAuth />);

      const signInButton = screen.getByText("Sign in with passkey");
      fireEvent.click(signInButton);

      await waitFor(() => {
        expect(
          screen.getByText("Authentication was cancelled or timed out.")
        ).toBeInTheDocument();
      });
    });

    it("should handle different error types correctly", async () => {
      const errorCases = [
        {
          error: "NotSupportedError",
          expectedMessage:
            "WebAuthn is not supported on this device or browser.",
        },
        {
          error: "SecurityError",
          expectedMessage:
            "Security error occurred. Please ensure you're on a secure connection.",
        },
        {
          error: "InvalidStateError",
          expectedMessage:
            "Invalid state. Please refresh the page and try again.",
        },
      ];

      for (const { error, expectedMessage } of errorCases) {
        mockSignIn.mockResolvedValue({
          error,
          url: null,
          ok: false,
          status: 401,
        });

        render(<WebAuthnAuth />);

        const signInButton = screen.getByText("Sign in with passkey");
        fireEvent.click(signInButton);

        await waitFor(() => {
          expect(screen.getByText(expectedMessage)).toBeInTheDocument();
        });
      }
    });
  });

  describe("UI Elements", () => {
    it("should render security features list", () => {
      render(<WebAuthnAuth />);

      expect(
        screen.getByText("Secured by WebAuthn standard")
      ).toBeInTheDocument();
      expect(
        screen.getByText(
          "• Works with Face ID, Touch ID, Windows Hello, or security keys"
        )
      ).toBeInTheDocument();
      expect(
        screen.getByText("• No passwords to remember or store")
      ).toBeInTheDocument();
      expect(
        screen.getByText("• Phishing-resistant authentication")
      ).toBeInTheDocument();
    });

    it("should disable buttons during loading", async () => {
      mockSignIn.mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(
              () =>
                resolve({
                  error: null,
                  url: "/dashboard",
                  ok: true,
                  status: 200,
                }),
              100
            )
          )
      );

      render(<WebAuthnAuth />);

      const signInButton = screen.getByText("Sign in with passkey");
      const createAccountButton = screen.getByText(
        "Create account with passkey"
      );

      fireEvent.click(signInButton);

      await waitFor(() => {
        expect(signInButton).toBeDisabled();
        expect(createAccountButton).toBeDisabled();
      });
    });

    it("should show appropriate loading text for different actions", async () => {
      mockSignIn.mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(
              () =>
                resolve({
                  error: null,
                  url: "/dashboard",
                  ok: true,
                  status: 200,
                }),
              100
            )
          )
      );

      render(<WebAuthnAuth />);

      // Test sign in loading
      const signInButton = screen.getByText("Sign in with passkey");
      fireEvent.click(signInButton);

      await waitFor(() => {
        expect(screen.getByText("Signing in...")).toBeInTheDocument();
      });

      // Reset and test create account loading
      vi.clearAllMocks();
      render(<WebAuthnAuth />);

      const createAccountButton = screen.getByText(
        "Create account with passkey"
      );
      fireEvent.click(createAccountButton);

      await waitFor(() => {
        expect(screen.getByText("Creating account...")).toBeInTheDocument();
      });
    });
  });
});
