"use client";

import Link from "next/link";
import { signOut } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Home, Key, Settings, LogOut, ArrowLeft } from "lucide-react";

export function DashboardNav() {
  const handleSignOut = () => {
    signOut({ callbackUrl: "/" });
  };

  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          {/* Left side - Logo and Navigation */}
          <div className="flex items-center space-x-8">
            {/* Back to main site */}
            <Link
              href="/"
              className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              <span className="text-sm font-medium">Back to Site</span>
            </Link>

            {/* Dashboard Title */}
            <div className="flex items-center space-x-2">
              <h1 className="text-xl font-semibold text-gray-900">Dashboard</h1>
            </div>

            {/* Navigation Links */}
            <div className="hidden md:flex items-center space-x-6">
              <Link
                href="/dashboard"
                className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
              >
                <Home className="h-4 w-4" />
                <span className="text-sm font-medium">Overview</span>
              </Link>

              <Link
                href="/dashboard/tokens"
                className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
              >
                <Key className="h-4 w-4" />
                <span className="text-sm font-medium">JWT Tokens</span>
              </Link>

              <Link
                href="/dashboard/sessions"
                className="flex items-center space-x-2 text-gray-600 hover:text-gray-900 transition-colors"
              >
                <Settings className="h-4 w-4" />
                <span className="text-sm font-medium">Sessions</span>
              </Link>
            </div>
          </div>

          {/* Right side - Sign Out */}
          <div className="flex items-center">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleSignOut}
              className="flex items-center space-x-2 text-gray-600 hover:text-gray-900"
            >
              <LogOut className="h-4 w-4" />
              <span className="hidden sm:inline">Sign Out</span>
            </Button>
          </div>
        </div>
      </div>
    </nav>
  );
}
