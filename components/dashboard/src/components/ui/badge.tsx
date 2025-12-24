import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "inline-flex items-center justify-center border px-3 py-1 text-[10px] tracking-wider uppercase font-light w-fit whitespace-nowrap shrink-0 [&>svg]:size-3 gap-1 [&>svg]:pointer-events-none transition-colors overflow-hidden",
  {
    variants: {
      variant: {
        default:
          "border-stone-800/90 bg-stone-100 text-stone-900 [a&]:hover:bg-stone-200 dark:border-stone-700/90 dark:bg-stone-800 dark:text-stone-300 dark:[a&]:hover:bg-stone-700",
        secondary:
          "border-stone-200 bg-stone-50 text-stone-600 [a&]:hover:bg-stone-100 dark:border-stone-600 dark:bg-stone-900 dark:text-stone-400 dark:[a&]:hover:bg-stone-800",
        destructive:
          "border-red-800/90 bg-red-100 text-red-900 [a&]:hover:bg-red-200 dark:border-red-700/90 dark:bg-red-900 dark:text-red-300 dark:[a&]:hover:bg-red-800",
        outline:
          "border-stone-800/90 text-stone-900 [a&]:hover:bg-stone-100 dark:border-stone-700/90 dark:text-stone-300 dark:[a&]:hover:bg-stone-800",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span"

  return (
    <Comp
      data-slot="badge"
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  )
}

export { Badge, badgeVariants }
