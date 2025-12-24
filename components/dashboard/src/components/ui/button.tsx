import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap text-[11px] tracking-wider uppercase font-light transition-all duration-300 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 outline-none",
  {
    variants: {
      variant: {
        default: "bg-gradient-to-r from-stone-800 to-stone-950 text-stone-50 hover:from-amber-900 hover:to-amber-950 shadow-warm-sm dark:from-stone-700 dark:to-stone-800 dark:hover:from-amber-600 dark:hover:to-orange-600 dark:shadow-night-sm",
        destructive:
          "bg-gradient-to-r from-red-800 to-red-950 text-stone-50 hover:from-red-900 hover:to-red-950 shadow-warm-sm dark:from-red-700 dark:to-red-800 dark:hover:from-red-600 dark:hover:to-red-700 dark:shadow-night-sm",
        outline:
          "border border-stone-800/90 bg-white/95 text-stone-900 hover:bg-stone-100 hover:text-amber-900 shadow-warm-sm dark:border-stone-700/90 dark:bg-stone-900/95 dark:text-stone-300 dark:hover:bg-stone-800 dark:hover:text-amber-500 dark:shadow-night-sm",
        secondary:
          "bg-stone-100 text-stone-900 hover:bg-amber-50 hover:text-amber-900 shadow-warm-sm dark:bg-stone-800 dark:text-stone-300 dark:hover:bg-stone-700 dark:hover:text-amber-500 dark:shadow-night-sm",
        ghost:
          "text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500",
        link: "text-stone-600 hover:text-amber-900 transition-colors dark:text-stone-400 dark:hover:text-amber-500",
      },
      size: {
        default: "h-12 px-4 py-2",
        sm: "h-10 px-3",
        lg: "h-14 px-6",
        icon: "size-12",
        "icon-sm": "size-10",
        "icon-lg": "size-14",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot : "button"

  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
