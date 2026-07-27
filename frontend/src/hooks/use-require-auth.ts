"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

export function useRequireAuth() {
  const router = useRouter();
  const { isAuthenticated, tokens, hasHydrated } = useAuthStore();

  useEffect(() => {
    if (!hasHydrated) return;
    if (!isAuthenticated || !tokens?.access_token) {
      router.replace("/login");
    }
  }, [hasHydrated, isAuthenticated, tokens, router]);

  return { isAuthenticated: hasHydrated ? isAuthenticated : true };
}
