import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva("inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-semibold leading-none", {
  variants: {
    variant: {
      neutral: "border-line bg-panel-raised text-muted",
      success: "border-accent/25 bg-accent/10 text-accent",
      warning: "border-warning/25 bg-warning/10 text-warning",
      danger: "border-danger/25 bg-danger/10 text-danger",
      info: "border-teal/25 bg-teal/10 text-teal",
    },
  },
  defaultVariants: { variant: "neutral" },
});

type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>;

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
