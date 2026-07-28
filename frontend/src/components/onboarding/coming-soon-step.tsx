"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import RotatingText from "@/components/ui/rotating-text";
import { useSaveOnboardingState, useCompleteOnboarding } from "@/queries/onboarding";
import { useReducedMotion } from "@/hooks/use-media-query";

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
  const prefersReducedMotion = useReducedMotion();

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
        {prefersReducedMotion ? (
          <CardTitle className="text-lg">{title}</CardTitle>
        ) : (
          <RotatingText
            texts={[title]}
            auto={false}
            splitBy="words"
            staggerDuration={0.03}
            mainClassName="text-lg font-medium leading-none"
            transition={{ type: "spring", damping: 25, stiffness: 300 }}
          />
        )}
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
