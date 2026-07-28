"use client";

import { Logo } from "@/components/brand/logo";
import RotatingText from "@/components/ui/rotating-text";
import { useReducedMotion } from "@/hooks/use-media-query";

const TAGLINES = [
  "Research sponsors faster.",
  "Close more partnerships.",
  "Automate your outreach.",
  "Know your pipeline.",
];

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const prefersReducedMotion = useReducedMotion();

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-muted/30 px-4 py-10">
      {/* Decorative marks — both light/dark nobg variants for brand presence */}
      <div
        aria-hidden
        className="pointer-events-none absolute -left-8 top-12 opacity-[0.07] dark:opacity-[0.12]"
      >
        <Logo size={160} style="mark" variant="black-nobg" className="dark:hidden" />
        <Logo size={160} style="mark" variant="white-nobg" className="hidden dark:block" />
      </div>
      <div
        aria-hidden
        className="pointer-events-none absolute -right-10 bottom-16 opacity-[0.07] dark:opacity-[0.12]"
      >
        <Logo size={180} style="mark" variant="black-nobg" className="dark:hidden" />
        <Logo size={180} style="mark" variant="white-nobg" className="hidden dark:block" />
      </div>

      <div className="relative z-10 mb-8">
        <Logo
          href="/login"
          size={40}
          style="solid"
          showWordmark
          priority
          wordmarkClassName="text-lg"
        />
      </div>

      <RotatingText
        texts={TAGLINES}
        mainClassName="relative z-10 mb-8 justify-center text-2xl font-semibold tracking-tight text-foreground"
        splitLevelClassName="overflow-hidden"
        elementLevelClassName="tracking-tight"
        splitBy="words"
        staggerDuration={0.02}
        rotationInterval={2800}
        transition={prefersReducedMotion ? { duration: 0 } : { type: "spring", damping: 25, stiffness: 300 }}
        auto={!prefersReducedMotion}
      />

      <div className="relative z-10 w-full max-w-md">{children}</div>
    </div>
  );
}
