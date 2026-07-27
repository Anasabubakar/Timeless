"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useOnboardingState } from "@/queries/onboarding";

export default function OnboardingIndexPage() {
  const router = useRouter();
  const { data, isLoading } = useOnboardingState();

  useEffect(() => {
    if (isLoading) return;
    const step = data?.data.step || "workspace";
    router.replace(`/onboarding/${step}`);
  }, [isLoading, data, router]);

  return (
    <div className="flex justify-center">
      <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
    </div>
  );
}
