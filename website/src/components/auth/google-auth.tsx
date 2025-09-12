"use client";

import { Loader2, Mail } from "lucide-react";
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

export function GoogleAuth() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleGoogleAuth = async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Use Auth.js signIn with Google provider
      const result = await signIn("google", {
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
      console.error("Google authentication error:", err);
      setError("Authentication failed. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const getErrorMessage = (error: string): string => {
    switch (error) {
      case "OAuthSignin":
        return "Error occurred during Google sign-in. Please try again.";
      case "OAuthCallback":
        return "Error occurred during Google callback. Please try again.";
      case "OAuthCreateAccount":
        return "Could not create account with Google. Please try again.";
      case "EmailCreateAccount":
        return "Could not create account with this email address.";
      case "Callback":
        return "Error occurred during authentication callback.";
      case "OAuthAccountNotLinked":
        return "This email is already associated with another account. Please sign in with your original method.";
      case "EmailSignin":
        return "Error occurred during email sign-in.";
      case "CredentialsSignin":
        return "Invalid credentials provided.";
      case "SessionRequired":
        return "Please sign in to access this page.";
      default:
        return `Authentication failed: ${error}. Please try again.`;
    }
  };

  return (
    <Card className="border-blue-200 dark:border-blue-800">
      <CardHeader className="pb-4">
        <CardTitle className="flex items-center space-x-2 text-lg">
          <div className="flex h-5 w-5 items-center justify-center rounded bg-blue-500">
            <Mail className="h-3 w-3 text-white" />
          </div>
          <span>Google Account</span>
        </CardTitle>
        <CardDescription>
          Sign in with your Google account for quick and secure access.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <Button
          onClick={handleGoogleAuth}
          disabled={isLoading}
          className="w-full bg-blue-600 hover:bg-blue-700 text-white"
          size="lg"
        >
          {isLoading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <svg
              className="mr-2 h-4 w-4"
              viewBox="0 0 24 24"
              aria-label="Google logo"
            >
              <title>Google</title>
              <path
                fill="currentColor"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="currentColor"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="currentColor"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="currentColor"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
          )}
          {isLoading ? "Signing in..." : "Continue with Google"}
        </Button>

        {error && (
          <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-md border border-destructive/20">
            <div className="font-medium">Authentication Error</div>
            <div className="mt-1">{error}</div>
          </div>
        )}

        <div className="text-xs text-muted-foreground space-y-1">
          <div>• Sign in with your existing Google account</div>
          <div>• Automatically creates an account if you're new</div>
          <div>• Secure OAuth 2.0 authentication</div>
          <div>• No passwords stored on our servers</div>
        </div>
      </CardContent>
    </Card>
  );
}
