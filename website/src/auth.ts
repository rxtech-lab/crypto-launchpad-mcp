import NextAuth from "next-auth";
import Google from "next-auth/providers/google";
import WebAuthn from "next-auth/providers/webauthn";
import { DrizzleAdapter } from "@auth/drizzle-adapter";
import { db } from "@/lib/db/connection";
import type { NextAuthConfig } from "next-auth";

// Use database sessions with Drizzle adapter
const config = {
  adapter: DrizzleAdapter(db),
  experimental: {
    enableWebAuthn: true,
  },
  providers: [
    WebAuthn({
      // Relying Party (RP) configuration
      name: "Crypto Launchpad",
      // Enable conditional UI for better UX
      enableConditionalUI: true,
    }),
    Google({
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
      // Allow account linking
      allowDangerousEmailAccountLinking: true,
    }),
  ],
  session: {
    strategy: "database", // Use database sessions for proper session management
    maxAge: 30 * 24 * 60 * 60, // 30 days
  },
  callbacks: {
    async session({ session, user }) {
      // Send properties to the client
      if (user && session.user) {
        session.user.id = user.id;
        session.user.email = user.email;
        session.user.name = user.name;
        session.user.image = user.image;
      }
      return session;
    },
    async signIn() {
      // Allow sign in for all providers
      return true;
    },
    async redirect({ url, baseUrl }) {
      // Allows relative callback URLs
      if (url.startsWith("/")) return `${baseUrl}${url}`;
      // Allows callback URLs on the same origin
      else if (new URL(url).origin === baseUrl) return url;
      return baseUrl;
    },
  },
  pages: {
    signIn: "/auth",
    error: "/auth/error",
  },
  events: {
    async createUser({ user }) {
      console.log("New user created:", user.id);
    },
    async signIn({ user, account }) {
      console.log("User signed in:", user.id, "Provider:", account?.provider);
    },
  },
  debug: process.env.NODE_ENV === "development",
} satisfies NextAuthConfig;

export const { handlers, auth, signIn, signOut } = NextAuth(config);
