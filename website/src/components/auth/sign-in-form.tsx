"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { GoogleAuth } from "./google-auth";
import { WebAuthnAuth } from "./webauthn-auth";

export function SignInForm() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">Welcome</CardTitle>
          <CardDescription>
            Sign in to your account or create a new one to access your dashboard
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center text-sm text-muted-foreground mb-6">
            Choose your preferred authentication method:
          </div>
        </CardContent>
      </Card>

      {/* WebAuthn Authentication Component */}
      <WebAuthnAuth />

      {/* Google OAuth Authentication Component */}
      <GoogleAuth />

      <div className="text-center text-xs text-muted-foreground">
        By signing in, you agree to our terms of service and privacy policy
      </div>
    </div>
  );
}
