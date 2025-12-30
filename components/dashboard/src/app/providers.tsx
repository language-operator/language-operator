'use client'

import { SessionProvider } from 'next-auth/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { Toaster } from 'sonner'
import { ThemeProvider } from '@/components/theme-provider'
import { NavigationLoadingProvider } from '@/contexts/navigation-loading-context'
import { NavigationProgress } from '@/components/ui/navigation-progress'
import { SidebarProvider } from '@/contexts/sidebar-context'

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 5 * 60 * 1000, // 5 minutes - data stays fresh longer since we have real-time updates
            retry: 3, // Retry failed requests
            retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000), // Exponential backoff
          },
        },
      })
  )

  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <SessionProvider>
        <QueryClientProvider client={queryClient}>
          <SidebarProvider>
            <NavigationLoadingProvider>
              <NavigationProgress />
              {children}
              <Toaster richColors position="top-right" />
            </NavigationLoadingProvider>
          </SidebarProvider>
        </QueryClientProvider>
      </SessionProvider>
    </ThemeProvider>
  )
}
