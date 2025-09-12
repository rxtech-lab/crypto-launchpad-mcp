"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";
import { User, LogIn } from "lucide-react";

export function SiteHeader() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading, user } = useAuth();

  // Don't render header on dashboard or auth pages (they have their own navigation)
  if (pathname.startsWith("/dashboard") || pathname.startsWith("/auth")) {
    return null;
  }

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-14 items-center justify-between px-4">
        {/* Logo/Brand */}
        <Link href="/" className="flex items-center space-x-2">
          <span className="font-bold text-lg">Crypto Launchpad</span>
        </Link>

        {/* Navigation Links */}
        <nav className="hidden md:flex items-center space-x-6">
          <Link
            href="#features"
            className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
          >
            Features
          </Link>
          <Link
            href="#installation"
            className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
          >
            Installation
          </Link>
          <Link
            href="https://github.com/rxtech-lab/crypto-launchpad-mcp"
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
          >
            GitHub
          </Link>
        </nav>

        {/* Dashboard Button */}
        <div className="flex items-center space-x-2">
          {isLoading ? (
            <Button variant="ghost" size="sm" disabled>
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
            </Button>
          ) : isAuthenticated ? (
            <Button asChild variant="default" size="sm">
              <Link href="/dashboard" className="flex items-center space-x-2">
                <User className="h-4 w-4" />
                <span className="hidden sm:inline">
                  {user?.name ? `${user.name}'s Dashboard` : "Dashboard"}
                </span>
                <span className="sm:hidden">Dashboard</span>
              </Link>
            </Button>
          ) : (
            <Button asChild variant="default" size="sm">
              <Link href="/auth" className="flex items-center space-x-2">
                <LogIn className="h-4 w-4" />
                <span className="hidden sm:inline">Sign In</span>
                <span className="sm:hidden">Sign In</span>
              </Link>
            </Button>
          )}
        </div>
      </div>
    </header>
  );
}
