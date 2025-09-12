"use client";

import { Fingerprint, Loader2 } from "lucide-react";
import { signIn } from "next-auth/react";
import { useState } from "react";
import { Button } from "@/components/ui/button";

interface WebAuthnButtonProps {
  mode?: "signin" | "signup";
  className?: string;
}

export function WebAuthnButton({
  mode = "signin",
  className,
}: WebAuthnButtonProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleWebAuthnAuth = async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Use Auth.js signIn with WebAuthn provider
      const result = await signIn("webauthn", {
        redirect: false,
        callbackUrl: "/dashboard",
      });

      if (result?.error) {
        setError(getErrorMessage(result.error));
      } else if (result?.url) {
        // Successful authentication, redirect will be handled by Auth.js
        window.location.href = result.url;
      }
    } catch (err) {
      console.error("WebAuthn authentication error:", err);
      setError("Authentication failed. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const getErrorMessage = (error: string): string => {
    switch (error) {
      case "NotAllowedError":
        return "Authentication was cancelled or timed out.";
      case "NotSupportedError":
        return "WebAuthn is not supported on this device.";
      case "SecurityError":
        return "Security error occurred. Please try again.";
      case "InvalidStateError":
        return "Invalid state. Please refresh and try again.";
      case "ConstraintError":
        return "Constraint error. Please try again.";
      case "NotReadableError":
        return "Could not read authenticator data.";
      default:
        return "Authentication failed. Please try again.";
    }
  };

  const buttonText =
    mode === "signup" ? "Create account with passkey" : "Sign in with passkey";

  const buttonDescription =
    mode === "signup"
      ? "Use your device's biometric authentication to create a secure account"
      : "Use your device's biometric authentication to sign in securely";

  return (
    <div className="space-y-2">
      <Button
        onClick={handleWebAuthnAuth}
        disabled={isLoading}
        className={`w-full ${className}`}
        variant="outline"
      >
        {isLoading ? (
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
        ) : (
          <Fingerprint className="mr-2 h-4 w-4" />
        )}
        {isLoading ? "Authenticating..." : buttonText}
      </Button>

      <p className="text-xs text-muted-foreground text-center">
        {buttonDescription}
      </p>

      {error && (
        <div className="text-sm text-destructive text-center bg-destructive/10 p-2 rounded-md">
          {error}
        </div>
      )}
    </div>
  );
}
