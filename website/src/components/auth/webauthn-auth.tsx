"use client";

import { Fingerprint, Loader2, Shield, Smartphone } from "lucide-react";
import { signIn } from "next-auth/react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export function WebAuthnAuth() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isRegistering, setIsRegistering] = useState(false);

  const handleWebAuthnFlow = async (isNewUser: boolean = false) => {
    try {
      setIsLoading(true);
      setError(null);
      setIsRegistering(isNewUser);

      // Use Auth.js signIn with WebAuthn provider
      // Auth.js will handle both registration and authentication automatically
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
      console.error("WebAuthn error:", err);
      setError("Authentication failed. Please try again.");
    } finally {
      setIsLoading(false);
      setIsRegistering(false);
    }
  };

  const getErrorMessage = (error: string): string => {
    switch (error) {
      case "NotAllowedError":
        return "Authentication was cancelled or timed out.";
      case "NotSupportedError":
        return "WebAuthn is not supported on this device or browser.";
      case "SecurityError":
        return "Security error occurred. Please ensure you're on a secure connection.";
      case "InvalidStateError":
        return "Invalid state. Please refresh the page and try again.";
      case "ConstraintError":
        return "Constraint error. Your authenticator may not support this operation.";
      case "NotReadableError":
        return "Could not read authenticator data. Please try again.";
      case "AbortError":
        return "Operation was aborted. Please try again.";
      default:
        return `Authentication failed: ${error}. Please try again.`;
    }
  };

  const isWebAuthnSupported = () => {
    return (
      typeof window !== "undefined" &&
      window.PublicKeyCredential &&
      typeof window.PublicKeyCredential === "function"
    );
  };

  if (!isWebAuthnSupported()) {
    return (
      <Card className="border-muted">
        <CardContent className="pt-6">
          <div className="flex items-center space-x-2 text-muted-foreground">
            <Shield className="h-4 w-4" />
            <span className="text-sm">
              WebAuthn is not supported on this device or browser.
            </span>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-primary/20">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center space-x-2 text-lg">
          <Fingerprint className="h-5 w-5 text-primary" />
          <span>Passkey Authentication</span>
        </CardTitle>
        <CardDescription>
          Use your device's biometric authentication or security key for secure,
          passwordless access.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-1 gap-3">
          <Button
            onClick={() => handleWebAuthnFlow(false)}
            disabled={isLoading}
            className="w-full"
            size="lg"
          >
            {isLoading && !isRegistering ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Fingerprint className="mr-2 h-4 w-4" />
            )}
            {isLoading && !isRegistering
              ? "Signing in..."
              : "Sign in with passkey"}
          </Button>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs uppercase">
              <span className="bg-background px-2 text-muted-foreground">
                or
              </span>
            </div>
          </div>

          <Button
            onClick={() => handleWebAuthnFlow(true)}
            disabled={isLoading}
            variant="outline"
            className="w-full"
            size="lg"
          >
            {isLoading && isRegistering ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Smartphone className="mr-2 h-4 w-4" />
            )}
            {isLoading && isRegistering
              ? "Creating account..."
              : "Create account with passkey"}
          </Button>
        </div>

        {error && (
          <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-md border border-destructive/20">
            <div className="font-medium">Authentication Error</div>
            <div className="mt-1">{error}</div>
          </div>
        )}

        <div className="text-xs text-muted-foreground space-y-1">
          <div className="flex items-center space-x-1">
            <Shield className="h-3 w-3" />
            <span>Secured by WebAuthn standard</span>
          </div>
          <div>
            • Works with Face ID, Touch ID, Windows Hello, or security keys
          </div>
          <div>• No passwords to remember or store</div>
          <div>• Phishing-resistant authentication</div>
        </div>
      </CardContent>
    </Card>
  );
}
