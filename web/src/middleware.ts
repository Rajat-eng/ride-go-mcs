import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Routes the middleware should NOT guard (auth + static assets handled by matcher)
const PUBLIC_PATHS = ['/auth/login', '/auth/signup', '/auth/callback'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public auth pages through
  if (PUBLIC_PATHS.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  // The refresh_token is an HttpOnly cookie set by the backend.
  // HttpOnly prevents client-side JS from reading it, but Next.js middleware
  // runs on the Edge (server-side) and receives it via the Cookie header.
  const refreshToken = request.cookies.get('refresh_token')?.value;

  if (!refreshToken) {
    // No active session — redirect to home where the login form is shown
    const loginUrl = new URL('/', request.url);
    loginUrl.searchParams.set('auth', 'required');
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * Guard all routes except:
     *  - / (home / login page)
     *  - _next/static, _next/image, favicon.ico (Next.js internals)
     *  - /api/* (API routes, if any)
     */
    '/((?!$|_next/static|_next/image|favicon.ico|api/).*)',
  ],
};
