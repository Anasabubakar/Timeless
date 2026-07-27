"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

export function OnboardingGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isAuthenticated, tokens, user, hasHydrated } = useAuthStore();

  useEffect(() => {
    if (!hasHydrated) return;
    if (!isAuthenticated || !tokens?.access_token) {
      router.replace("/login");
      return;
    }
    if (user?.onboarding_completed) {
      router.replace("/dashboard");
    }
  }, [hasHydrated, isAuthenticated, tokens, user, router]);

  if (!hasHydrated) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
      </div>
    );
  }

  if (!isAuthenticated || !tokens?.access_token || user?.onboarding_completed) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
