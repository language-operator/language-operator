'use client'

import { useSession } from 'next-auth/react'
import { useRouter } from 'next/navigation'
import { useEffect } from 'react'
import { Sidebar } from './sidebar'
import { Header } from './header'
import { ClusterProvider } from '@/contexts/cluster-context'
import { useOrganizations } from '@/hooks/use-organizations'

export function AuthenticatedLayout({ children }: { children: React.ReactNode }) {
  const { data: session, status } = useSession()
  const router = useRouter()
  
  // Load organizations on app startup to initialize active organization
  useOrganizations()

  useEffect(() => {
    if (status === 'unauthenticated') {
      router.push('/login')
    }
  }, [status, router])

  if (status === 'loading') {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-lg">Loading...</div>
      </div>
    )
  }

  if (!session) {
    return null
  }

  return (
    <ClusterProvider>
      <div className="flex h-screen overflow-hidden bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950">
        <Sidebar />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header />
          <main className="flex-1 overflow-y-auto p-generous">
            {children}
          </main>
        </div>
      </div>
    </ClusterProvider>
  )
}
