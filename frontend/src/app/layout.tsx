import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "./providers";
import { Toaster } from "sonner";

export const metadata: Metadata = {
  title: "SponsorOS — Sponsorship Intelligence Platform",
  description: "AI-powered sponsorship intelligence and operations platform",
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
