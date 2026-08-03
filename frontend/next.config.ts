import type { NextConfig } from "next";

// No security headers were configured at all — every response (the
// actual dashboard HTML, not just the API) shipped with no
// X-Frame-Options, no CSP, nothing stopping this app from being framed
// by another site. script-src/style-src keep 'unsafe-inline' (and
// 'unsafe-eval' only in dev, for Turbopack HMR) since Next's App Router
// injects inline hydration scripts without a nonce by default —
// tightening that further needs per-request nonces wired through
// middleware, which is a real follow-up, not a drop-in change here.
const isDev = process.env.NODE_ENV !== "production";
const apiOrigin = (() => {
  try {
    return new URL(process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").origin;
  } catch {
    return "http://localhost:8080";
  }
})();
// The realtime WebSocket connects to the same host as apiOrigin (see
// useWebSocket, which derives its URL from NEXT_PUBLIC_API_URL) — CSP
// treats ws(s):// and http(s):// to the same host as distinct connect-src
// entries, so listing only the https origin here silently blocks the
// WebSocket regardless of what URL the app code actually points at.
const wsOrigin = apiOrigin.replace(/^http/, "ws");

const securityHeaders = [
  { key: "X-Frame-Options", value: "DENY" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy", value: "geolocation=(), microphone=(), camera=(), payment=()" },
  {
    key: "Content-Security-Policy",
    value: [
      "default-src 'self'",
      `script-src 'self' 'unsafe-inline'${isDev ? " 'unsafe-eval'" : ""}`,
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: https:",
      "font-src 'self' data:",
      `connect-src 'self' ${apiOrigin} ${wsOrigin}${isDev ? " ws://localhost:*" : ""}`,
      "object-src 'none'",
      "base-uri 'self'",
      "frame-ancestors 'none'",
    ].join("; "),
  },
];

const nextConfig: NextConfig = {
  experimental: {
    optimizePackageImports: ["lucide-react", "recharts"],
  },
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default nextConfig;
