import { type NextRequest, NextResponse } from "next/server";
import { getToken } from "next-auth/jwt";

const isDevelopmentEnvironment = process.env.NODE_ENV === "development";

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  /** Playwright starts the dev server and requires a 200 status to
   * begin the tests, so this ensures that the tests can start
   */
  if (pathname.startsWith("/ping")) {
    return new Response("pong", { status: 200 });
  }

  if (pathname.startsWith("/api/auth")) {
    return NextResponse.next();
  }

  if (pathname.startsWith("/api/workflow")) {
    return NextResponse.next();
  }

  const token = await getToken({
    req: request,
    secret: process.env.AUTH_SECRET,
    secureCookie: !isDevelopmentEnvironment,
  });

  // Define protected routes
  const isProtectedRoute = pathname.startsWith("/dashboard");

  // Define auth routes
  const isAuthRoute = pathname.startsWith("/auth");

  // If no token and trying to access protected route, redirect to auth
  if (!token && isProtectedRoute) {
    return NextResponse.redirect(new URL("/auth", request.url));
  }

  // If token exists and trying to access auth pages, redirect to dashboard
  if (token && isAuthRoute) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/",
    "/dashboard/:path*",
    "/auth/:path*",
    "/api/:path*",
    /**
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico, sitemap.xml, robots.txt (metadata files)
     */
    "/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)",
  ],
};
