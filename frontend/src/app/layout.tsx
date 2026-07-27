import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Providers } from "./providers";
import { Toaster } from "sonner";

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export const metadata: Metadata = {
  title: "Timeless — Sponsorship Intelligence Platform",
  description: "AI-powered sponsorship intelligence and operations platform",
  icons: {
    icon: [
      { url: "/images/logo-black.svg", type: "image/svg+xml", media: "(prefers-color-scheme: light)" },
      { url: "/images/logo-white.svg", type: "image/svg+xml", media: "(prefers-color-scheme: dark)" },
      { url: "/images/logo-black.svg", type: "image/svg+xml" },
      { url: "/favicon.ico", sizes: "32x32" },
    ],
    shortcut: "/images/logo-black.svg",
    apple: "/images/logo-black.svg",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen font-sans">
        <Providers>
          {children}
          <Toaster position="bottom-right" richColors />
        </Providers>
      </body>
    </html>
  );
}
