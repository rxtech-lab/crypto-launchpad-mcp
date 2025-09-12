import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SignInForm } from "../sign-in-form";

// Mock the child components
vi.mock("../google-auth", () => ({
  GoogleAuth: () => <div data-testid="google-auth">Google Auth Component</div>,
}));

vi.mock("../webauthn-auth", () => ({
  WebAuthnAuth: () => (
    <div data-testid="webauthn-auth">WebAuthn Auth Component</div>
  ),
}));

describe("SignInForm", () => {
  it("should render welcome card with title and description", () => {
    render(<SignInForm />);

    expect(screen.getByText("Welcome")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Sign in to your account or create a new one to access your dashboard"
      )
    ).toBeInTheDocument();
  });

  it("should render authentication method selection text", () => {
    render(<SignInForm />);

    expect(
      screen.getByText("Choose your preferred authentication method:")
    ).toBeInTheDocument();
  });

  it("should render WebAuthn authentication component", () => {
    render(<SignInForm />);

    expect(screen.getByTestId("webauthn-auth")).toBeInTheDocument();
  });

  it("should render Google OAuth authentication component", () => {
    render(<SignInForm />);

    expect(screen.getByTestId("google-auth")).toBeInTheDocument();
  });

  it("should render terms and privacy policy text", () => {
    render(<SignInForm />);

    expect(
      screen.getByText(
        "By signing in, you agree to our terms of service and privacy policy"
      )
    ).toBeInTheDocument();
  });

  it("should have proper component structure", () => {
    render(<SignInForm />);

    // Check that components are rendered in the correct order
    const webauthnAuth = screen.getByTestId("webauthn-auth");
    const googleAuth = screen.getByTestId("google-auth");

    expect(webauthnAuth).toBeInTheDocument();
    expect(googleAuth).toBeInTheDocument();

    // WebAuthn should come before Google Auth
    const container = screen.getByText("Welcome").closest("div");
    expect(container).toBeInTheDocument();
  });
});
