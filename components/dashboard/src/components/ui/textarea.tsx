import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "w-full min-h-[96px] border border-stone-200 px-4 py-3 text-sm font-light bg-stone-50/30 text-stone-900 resize-none field-sizing-content",
        "focus:outline-none focus:border-amber-900/40 focus:ring-1 focus:ring-amber-900/20 transition-all",
        "dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:focus:border-amber-600/60 dark:focus:ring-amber-600/30",
        "placeholder:text-stone-500 dark:placeholder:text-stone-500",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
