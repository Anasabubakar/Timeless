"use client";

import Image from "next/image";
import Link from "next/link";
import { useTheme } from "next-themes";
import { useEffect, useState } from "react";
import { cn } from "@/lib/utils";

/**
 * Timeless logo assets in /public/images:
 * - logo-black.svg       solid black plate + white mark (light UI)
 * - logo-white.svg       solid white plate + dark mark (dark UI)
 * - logo-black-nobg.svg  black mark only (on light surfaces)
 * - logo-white-nobg.svg  white mark only (on dark surfaces)
 */
export const LOGO_ASSETS = {
  black: "/images/logo-black.svg",
  white: "/images/logo-white.svg",
  "black-nobg": "/images/logo-black-nobg.svg",
  "white-nobg": "/images/logo-white-nobg.svg",
} as const;

export type LogoAsset = keyof typeof LOGO_ASSETS;

export type LogoStyle = "solid" | "mark";

export interface LogoProps {
  className?: string;
  /** Pixel size of the mark / plate (square). */
  size?: number;
  /**
   * solid = plate variants (black/white backgrounds)
   * mark  = transparent nobg marks only
   */
  style?: LogoStyle;
  /**
   * Force a specific asset. "auto" picks by theme + style.
   */
  variant?: LogoAsset | "auto";
  /** Render the Timeless wordmark next to the mark. */
  showWordmark?: boolean;
  wordmarkClassName?: string;
  /** Wrap in a link to home/dashboard. */
  href?: string;
  priority?: boolean;
  alt?: string;
}

function resolveAsset(
  variant: LogoAsset | "auto",
  style: LogoStyle,
  isDark: boolean
): LogoAsset {
  if (variant !== "auto") return variant;
  if (style === "mark") {
    return isDark ? "white-nobg" : "black-nobg";
  }
  return isDark ? "white" : "black";
}

export function Logo({
  className,
  size = 28,
  style = "solid",
  variant = "auto",
  showWordmark = false,
  wordmarkClassName,
  href,
  priority = false,
  alt = "Timeless",
}: LogoProps) {
  const { resolvedTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => setMounted(true), []);

  // Avoid hydration mismatch: default to light asset until mounted.
  const isDark = mounted && resolvedTheme === "dark";
  const asset = resolveAsset(variant, style, isDark);
  const src = LOGO_ASSETS[asset];

  // nobg marks are not square (mark only); give a bit more width for the path.
  const isMark = style === "mark" || asset.endsWith("-nobg");
  const width = isMark ? Math.round(size * 1.2) : size;
  const height = size;

  const mark = (
    <Image
      src={src}
      alt={alt}
      width={width}
      height={height}
      priority={priority}
      className={cn("shrink-0 object-contain", className)}
      // SVGs from public — avoid optimization edge cases
      unoptimized
    />
  );

  const content = (
    <span className={cn("inline-flex items-center gap-2", showWordmark && "min-w-0")}>
      {mark}
      {showWordmark && (
        <span
          className={cn(
            "text-sm font-semibold tracking-tight text-foreground",
            wordmarkClassName
          )}
        >
          Timeless
        </span>
      )}
    </span>
  );

  if (href) {
    return (
      <Link href={href} className="inline-flex items-center outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-md">
        {content}
      </Link>
    );
  }

  return content;
}

/** Renders every logo variant — useful for brand kits / about surfaces. */
export function LogoGallery({ className }: { className?: string }) {
  return (
    <div className={cn("grid grid-cols-2 gap-4 sm:grid-cols-4", className)}>
      {(Object.keys(LOGO_ASSETS) as LogoAsset[]).map((key) => (
        <div
          key={key}
          className={cn(
            "flex flex-col items-center gap-2 rounded-xl border border-border p-4",
            key.includes("white") ? "bg-neutral-950" : "bg-white"
          )}
        >
          <Image
            src={LOGO_ASSETS[key]}
            alt={`Timeless ${key}`}
            width={48}
            height={48}
            unoptimized
            className="object-contain"
          />
          <span
            className={cn(
              "text-[10px] font-medium",
              key.includes("white") ? "text-neutral-400" : "text-neutral-500"
            )}
          >
            {key}
          </span>
        </div>
      ))}
    </div>
  );
}
