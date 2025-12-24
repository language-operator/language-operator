import * as React from "react"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "w-full h-9 border border-stone-200 px-4 text-sm font-light bg-stone-50/30 text-stone-900",
        "focus:outline-none focus:border-amber-900/40 focus:ring-1 focus:ring-amber-900/20 transition-all",
        "dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:focus:border-amber-600/60 dark:focus:ring-amber-600/30",
        "placeholder:text-stone-500 dark:placeholder:text-stone-500",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        "file:border-0 file:bg-transparent file:text-sm file:font-light file:text-stone-600 dark:file:text-stone-400",
        className
      )}
      {...props}
    />
  )
}

export { Input }
