"use client";

import * as React from "react";
import { AnimatePresence, motion } from "motion/react";
import { cn } from "@/lib/utils";
import { useIsMobile, useReducedMotion } from "@/hooks/use-media-query";

interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}

// Renders as a centered modal on desktop/tablet, and a swipeable bottom
// sheet on phones — every dialog in the app gets this for free since they
// all go through this one component.
export function Dialog({ open, onOpenChange, children }: DialogProps) {
  const isMobile = useIsMobile();
  const prefersReducedMotion = useReducedMotion();

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-[55] flex items-end justify-center sm:items-center">
          <motion.div
            className="fixed inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => onOpenChange(false)}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: prefersReducedMotion ? 0 : 0.2 }}
          />
          {isMobile ? (
            <motion.div
              className="relative z-50 max-h-[85vh] w-full overflow-y-auto rounded-t-2xl pb-safe"
              initial={{ y: "100%" }}
              animate={{ y: 0 }}
              exit={{ y: "100%" }}
              transition={prefersReducedMotion ? { duration: 0 } : { type: "spring", damping: 32, stiffness: 320 }}
              drag="y"
              dragConstraints={{ top: 0, bottom: 0 }}
              dragElastic={{ top: 0, bottom: 0.5 }}
              onDragEnd={(_, info) => {
                if (info.offset.y > 120 || info.velocity.y > 500) {
                  onOpenChange(false);
                }
              }}
            >
              <div className="mx-auto mb-1 mt-2 h-1.5 w-10 shrink-0 rounded-full bg-border" aria-hidden />
              {children}
            </motion.div>
          ) : (
            <motion.div
              className="relative z-50 w-full max-w-lg px-4 sm:px-0"
              initial={{ opacity: 0, scale: 0.96, y: 8 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.96, y: 8 }}
              transition={prefersReducedMotion ? { duration: 0 } : { type: "spring", damping: 28, stiffness: 340 }}
            >
              {children}
            </motion.div>
          )}
        </div>
      )}
    </AnimatePresence>
  );
}

export function DialogContent({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "rounded-t-2xl border border-neutral-200 bg-white p-5 shadow-xl dark:border-neutral-800 dark:bg-neutral-900 sm:rounded-xl sm:p-6",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export function DialogHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-col space-y-1.5 text-center sm:text-left", className)}
      {...props}
    />
  );
}

export function DialogTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn("text-lg font-semibold leading-none tracking-tight", className)}
      {...props}
    />
  );
}

export function DialogDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("text-sm text-neutral-500 dark:text-neutral-400", className)} {...props} />;
}

export function DialogFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col-reverse gap-2 pt-4 [&>button]:w-full sm:flex-row sm:justify-end sm:gap-0 sm:space-x-2 sm:[&>button]:w-auto",
        className
      )}
      {...props}
    />
  );
}
