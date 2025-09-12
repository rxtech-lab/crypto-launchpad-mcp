import { AlertTriangle } from "lucide-react";
import { Suspense } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

function AuthErrorContent() {
  return (
    <Card>
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-6 w-6 text-destructive" />
        </div>
        <CardTitle className="text-2xl font-bold text-destructive">
          Authentication Error
        </CardTitle>
        <CardDescription>
          There was an error during the authentication process. This could be
          due to:
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <ul className="text-sm text-muted-foreground space-y-2">
          <li>• Network connectivity issues</li>
          <li>• Invalid credentials or expired session</li>
          <li>• Authentication provider temporarily unavailable</li>
          <li>• Browser security settings blocking the request</li>
        </ul>

        <div className="pt-4">
          <Button asChild className="w-full">
            <a href="/auth">Try Again</a>
          </Button>
        </div>

        <div className="text-center">
          <Button variant="ghost" asChild>
            <a href="/">Return to Home</a>
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

export default function AuthErrorPage() {
  return (
    <Suspense
      fallback={
        <Card>
          <CardContent className="flex items-center justify-center py-8">
            <div className="text-muted-foreground">Loading...</div>
          </CardContent>
        </Card>
      }
    >
      <AuthErrorContent />
    </Suspense>
  );
}
