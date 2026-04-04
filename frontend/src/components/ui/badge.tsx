import React from "react";
import { cn } from "@/lib/utils";

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: "default" | "secondary" | "outline" | "destructive";
}

export function Badge({ className, variant = "secondary", ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-[var(--radius-sm)] px-1.5 py-0.5 text-2xs font-semibold",
        {
          "bg-[var(--accent)] text-[var(--accent-fg)]": variant === "default",
          "bg-[var(--surface-secondary)] text-[var(--fg-secondary)]": variant === "secondary",
          "border border-[var(--border-color)] text-[var(--fg)]": variant === "outline",
          "bg-[var(--danger)] text-white": variant === "destructive",
        },
        className
      )}
      {...props}
    />
  );
}
