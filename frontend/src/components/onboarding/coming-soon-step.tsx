"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { useSaveOnboardingState, useCompleteOnboarding } from "@/queries/onboarding";

interface ComingSoonStepProps {
  step: string;
  title: string;
  description: string;
  nextStep?: string;
}

// Steps 2-4 of onboarding (AI discovery, goals, dashboard init) ship in a
// later phase. This keeps the wizard navigable end-to-end today: a user can
// always move forward and finish onboarding rather than hitting a dead end.
export function ComingSoonStep({ step, title, description, nextStep }: ComingSoonStepProps) {
  const router = useRouter();
  const saveState = useSaveOnboardingState();
  const completeOnboarding = useCompleteOnboarding();

  async function handleContinue() {
    if (nextStep) {
      await saveState.mutateAsync({ step: nextStep, payload: {} });
      router.push(`/onboarding/${nextStep}`);
    } else {
      await completeOnboarding.mutateAsync();
      router.push("/dashboard");
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          This step is coming soon. For now you can jump straight to your dashboard — your workspace
          will keep syncing in the background.
        </p>
        <Button
          onClick={handleContinue}
          disabled={saveState.isPending || completeOnboarding.isPending}
          className="w-full"
        >
          {nextStep ? "Continue" : "Go to dashboard"}
        </Button>
      </CardContent>
    </Card>
  );
}
