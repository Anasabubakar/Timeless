"use client";

import { usePathname } from "next/navigation";
import { motion } from "motion/react";
import { OnboardingGuard } from "@/components/onboarding-guard";
import { cn } from "@/lib/utils";

const STEPS = [
  { key: "workspace", label: "Connect", path: "/onboarding/workspace" },
  { key: "discovery", label: "Discover", path: "/onboarding/discovery" },
  { key: "goals", label: "Goals", path: "/onboarding/goals" },
  { key: "dashboard", label: "Dashboard", path: "/onboarding/dashboard" },
];

export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const activeIndex = Math.max(
    0,
    STEPS.findIndex((step) => pathname?.startsWith(step.path))
  );

  return (
    <OnboardingGuard>
      <div className="flex min-h-screen flex-col bg-background">
        <header className="border-b border-border px-8 py-5">
          <div className="mx-auto flex max-w-2xl items-center justify-between">
            {STEPS.map((step, index) => (
              <div key={step.key} className="flex flex-1 items-center last:flex-none">
                <div className="flex items-center gap-2">
                  <div
                    className={cn(
                      "flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium transition-colors",
                      index < activeIndex && "bg-primary text-primary-foreground",
                      index === activeIndex && "bg-primary text-primary-foreground",
                      index > activeIndex && "border border-border text-muted-foreground"
                    )}
                  >
                    {index + 1}
                  </div>
                  <span
                    className={cn(
                      "text-sm font-medium transition-colors",
                      index <= activeIndex ? "text-foreground" : "text-muted-foreground"
                    )}
                  >
                    {step.label}
                  </span>
                </div>
                {index < STEPS.length - 1 && (
                  <div className="relative mx-3 h-px flex-1 bg-border">
                    <motion.div
                      className="absolute inset-y-0 left-0 bg-primary"
                      initial={false}
                      animate={{ width: index < activeIndex ? "100%" : "0%" }}
                      transition={{ duration: 0.4, ease: "easeOut" }}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        </header>

        <main className="flex flex-1 items-center justify-center px-6 py-12">
          <motion.div
            key={pathname}
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -12 }}
            transition={{ duration: 0.35, ease: "easeOut" }}
            className="w-full max-w-xl"
          >
            {children}
          </motion.div>
        </main>
      </div>
    </OnboardingGuard>
  );
}
