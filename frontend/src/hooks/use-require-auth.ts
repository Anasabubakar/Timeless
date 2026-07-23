"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";

export function useRequireAuth() {
  const router = useRouter();
  const { isAuthenticated, tokens } = useAuthStore();

  useEffect(() => {
    if (!isAuthenticated || !tokens?.access_token) {
      router.replace("/login");
    }
  }, [isAuthenticated, tokens, router]);

  return { isAuthenticated };
}
