'use client'

import { useSession } from 'next-auth/react'
import { useRouter, useSearchParams, useParams } from 'next/navigation'
import { useEffect, Suspense } from 'react'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { ClusterProvider } from '@/contexts/cluster-context'
import { ConsoleLayout } from '@/components/console/console-layout'
import { ConsoleProvider } from '@/contexts/console-context'
import { useOrganizations } from '@/hooks/use-organizations'

function ConsoleContent({ clusterName }: { clusterName: string }) {
  const searchParams = useSearchParams()

  // Extract URL parameters for initial agent selection
  const agentParam = searchParams.get('agent')

  return (
    <ConsoleProvider initialAgent={agentParam} initialCluster={clusterName}>
      <ConsoleLayout />
    </ConsoleProvider>
  )
}

export default function ConsolePage() {
  const { data: session, status } = useSession()
  const router = useRouter()
  const params = useParams()
  const clusterName = params.name as string

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
          <Suspense fallback={<div className="flex h-full items-center justify-center"><div className="text-sm">Loading console...</div></div>}>
            <ConsoleContent clusterName={clusterName} />
          </Suspense>
        </div>
      </div>
    </ClusterProvider>
  )
}
