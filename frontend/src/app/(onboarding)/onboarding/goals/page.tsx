"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import RotatingText from "@/components/ui/rotating-text";
import {
  useRecommendGoals,
  usePlanAutomation,
  useApproveAutomation,
  type RecommendedGoal,
  type PlannedAutomation,
} from "@/queries/discovery";
import { useOnboardingState, useSaveOnboardingState, useCompleteOnboarding } from "@/queries/onboarding";
import { useReducedMotion } from "@/hooks/use-media-query";

export default function GoalsStepPage() {
  const router = useRouter();
  const prefersReducedMotion = useReducedMotion();
  const { data: onboardingState } = useOnboardingState();
  const recommendGoals = useRecommendGoals();
  const planAutomation = usePlanAutomation();
  const approveAutomation = useApproveAutomation();
  const saveState = useSaveOnboardingState();
  const completeOnboarding = useCompleteOnboarding();

  const [goals, setGoals] = useState<RecommendedGoal[]>([]);
  const [selectedGoal, setSelectedGoal] = useState<string>("");
  const [customGoal, setCustomGoal] = useState("");
  const [plan, setPlan] = useState<PlannedAutomation[] | null>(null);
  const [ran, setRan] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const projectNames = ((onboardingState?.data.payload as any)?.goals?.project_names || []) as string[];

  useEffect(() => {
    if (ran || !onboardingState) return;
    setRan(true);
    recommendGoals
      .mutateAsync({ project_names: projectNames })
      .then((res) => setGoals(res.data))
      .catch((err: any) => setError(err.message || "Goal recommendation failed"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ran, onboardingState]);

  async function handlePlan(goal: string) {
    setError(null);
    setSelectedGoal(goal);
    setPlan(null);
    try {
      const res = await planAutomation.mutateAsync({ goal });
      setPlan(res.data);
    } catch (err: any) {
      setError(err.message || "Automation planning failed");
    }
  }

  async function finishOnboarding() {
    await saveState.mutateAsync({ step: "dashboard", payload: {} });
    await completeOnboarding.mutateAsync();
    router.push("/dashboard");
  }

  async function handleApprove() {
    if (!plan) return;
    await approveAutomation.mutateAsync({ steps: plan });
    await finishOnboarding();
  }

  async function handleSkip() {
    await finishOnboarding();
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        {prefersReducedMotion ? (
          <h1 className="text-2xl font-semibold">What would you like to accomplish?</h1>
        ) : (
          <RotatingText
            texts={["What would you like to accomplish?"]}
            auto={false}
            splitBy="words"
            staggerDuration={0.03}
            mainClassName="justify-center text-2xl font-semibold"
            transition={{ type: "spring", damping: 25, stiffness: 300 }}
          />
        )}
        <p className="mt-2 text-sm text-muted-foreground">
          Pick a goal and I'll propose an automation plan for it.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {recommendGoals.isPending && (
        <div className="flex justify-center py-8">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
        </div>
      )}

      {!selectedGoal && (
        <>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {goals.map((goal, i) => (
              <motion.div
                key={goal.title}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, delay: i * 0.05 }}
              >
                <Card role="button" onClick={() => handlePlan(goal.title)} className="h-full cursor-pointer hover:border-primary">
                  <CardHeader>
                    <CardTitle className="text-sm">{goal.title}</CardTitle>
                    <CardDescription>{goal.description}</CardDescription>
                  </CardHeader>
                </Card>
              </motion.div>
            ))}
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium">Or type your own goal</label>
            <div className="flex gap-2">
              <Input value={customGoal} onChange={(e) => setCustomGoal(e.target.value)} placeholder="e.g. Grow renewals" />
              <Button disabled={!customGoal.trim()} onClick={() => handlePlan(customGoal.trim())}>
                Go
              </Button>
            </div>
          </div>
        </>
      )}

      {selectedGoal && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Automation plan for "{selectedGoal}"</CardTitle>
            <CardDescription>Here's how I'll automate it. Approve, or skip and set it up later.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {planAutomation.isPending && (
              <div className="flex justify-center py-4">
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
              </div>
            )}
            {plan?.map((step) => (
              <div key={step.title} className="rounded-lg border border-border p-3">
                <p className="text-sm font-medium">{step.title}</p>
                <p className="mt-1 text-xs text-muted-foreground">{step.description}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      <div className="flex items-center justify-between pt-2">
        <Button variant="ghost" onClick={handleSkip} disabled={completeOnboarding.isPending}>
          {selectedGoal ? "I'll do this later" : "Skip for now"}
        </Button>
        {selectedGoal && plan && (
          <Button onClick={handleApprove} disabled={approveAutomation.isPending || completeOnboarding.isPending}>
            Approve and finish
          </Button>
        )}
      </div>
    </div>
  );
}
