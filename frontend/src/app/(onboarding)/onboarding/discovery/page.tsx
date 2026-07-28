"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import RotatingText from "@/components/ui/rotating-text";
import { useRunDiscovery, useSelectProjects, type DiscoveredProject } from "@/queries/discovery";
import { useSaveOnboardingState } from "@/queries/onboarding";
import { useReducedMotion } from "@/hooks/use-media-query";

export default function DiscoveryStepPage() {
  const router = useRouter();
  const prefersReducedMotion = useReducedMotion();
  const runDiscovery = useRunDiscovery();
  const selectProjects = useSelectProjects();
  const saveState = useSaveOnboardingState();

  const [projects, setProjects] = useState<DiscoveredProject[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [newProject, setNewProject] = useState("");
  const [ran, setRan] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (ran) return;
    setRan(true);
    runDiscovery
      .mutateAsync()
      .then((res) => setProjects(res.data))
      .catch((err: any) => setError(err.message || "Discovery failed"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ran]);

  function toggle(name: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }

  async function handleContinue() {
    const names = Array.from(selected);
    await selectProjects.mutateAsync({ project_names: names, new_project: newProject.trim() || undefined });
    const allNames = newProject.trim() ? [...names, newProject.trim()] : names;
    await saveState.mutateAsync({ step: "goals", payload: { project_names: allNames } });
    router.push("/onboarding/goals");
  }

  async function handleSkip() {
    await saveState.mutateAsync({ step: "goals", payload: { project_names: [] } });
    router.push("/onboarding/goals");
  }

  const canContinue = selected.size > 0 || newProject.trim().length > 0;

  return (
    <div className="space-y-6">
      <div className="text-center">
        {prefersReducedMotion ? (
          <h1 className="text-2xl font-semibold">What would you like Timeless to focus on?</h1>
        ) : (
          <RotatingText
            texts={["What would you like Timeless to focus on?"]}
            auto={false}
            splitBy="words"
            staggerDuration={0.03}
            mainClassName="justify-center text-2xl font-semibold"
            transition={{ type: "spring", damping: 25, stiffness: 300 }}
          />
        )}
        <p className="mt-2 text-sm text-muted-foreground">
          I inspected your connected data and inferred what you might be working on.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {runDiscovery.isPending && (
        <div className="flex justify-center py-8">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-neutral-900 border-t-transparent" />
        </div>
      )}

      {!runDiscovery.isPending && projects.length === 0 && !error && (
        <Card>
          <CardContent className="py-6 text-center text-sm text-muted-foreground">
            Nothing detected yet — that's fine, your workspace is still syncing. Create a workspace
            below or skip for now.
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {projects.map((project, i) => (
          <motion.div
            key={project.name}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, delay: i * 0.05 }}
          >
            <Card
              className={selected.has(project.name) ? "border-primary" : undefined}
              onClick={() => toggle(project.name)}
              role="button"
            >
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{project.name}</CardTitle>
                  <Badge variant={project.confidence >= 70 ? "default" : "secondary"}>
                    {project.confidence}% confidence
                  </Badge>
                </div>
                <CardDescription>{project.explanation}</CardDescription>
              </CardHeader>
              <CardContent className="flex items-center gap-4 text-xs text-muted-foreground">
                <span>{project.document_count} related items</span>
                <span>{project.recent_activity}</span>
                <span>{project.sources?.join(", ")}</span>
              </CardContent>
            </Card>
          </motion.div>
        ))}
      </div>

      <div className="space-y-1.5">
        <label className="text-sm font-medium">Or create a new workspace</label>
        <Input
          value={newProject}
          onChange={(e) => setNewProject(e.target.value)}
          placeholder="e.g. Q3 Partnerships"
        />
      </div>

      <div className="flex items-center justify-between pt-2">
        <Button variant="ghost" onClick={handleSkip}>
          Skip
        </Button>
        <Button onClick={handleContinue} disabled={!canContinue || selectProjects.isPending}>
          Continue
        </Button>
      </div>
    </div>
  );
}
