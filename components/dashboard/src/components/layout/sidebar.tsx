'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { useClusterContext } from '@/contexts/cluster-context'
import { ClusterSelector } from '@/components/cluster-selector'
import {
  Home,
  Bot,
  Cpu,
  Wrench,
  Users,
  Cloud,
  Settings,
  BarChart3,
} from 'lucide-react'

const globalNavigation = [
  { name: 'Overview', href: '/', icon: Home },
  { name: 'Clusters', href: '/clusters', icon: Cloud },
]

const clusterNavigation = [
  { name: 'Dashboard', href: '', icon: BarChart3 }, // Will be prefixed with /clusters/[name]
  { name: 'Agents', href: '/agents', icon: Bot },
  { name: 'Tools', href: '/tools', icon: Wrench },
  { name: 'Personas', href: '/personas', icon: Users },
  { name: 'Models', href: '/models', icon: Cpu },
]

export function Sidebar() {
  const pathname = usePathname()
  const { selectedCluster, isClusterSelected } = useClusterContext()

  return (
    <div className="flex h-screen w-64 flex-col border-r bg-gray-50 dark:bg-gray-900">
      <div className="flex h-16 items-center border-b px-3">
        <div className="px-3 py-2 font-mono text-sm w-full">
          <span className="text-black dark:text-white font-bold tracking-wide">
            LANGUAGE OPERATOR{' '}
            <span className="animate-pulse">█</span>
          </span>
        </div>
      </div>
      
      {/* Cluster Selector */}
      <ClusterSelector />
      
      <nav className="flex-1 space-y-1 px-3 py-4">
        {/* Global Navigation */}
        <div className="mb-4">
          <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider px-3 pb-2">
            Global
          </div>
          {globalNavigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white'
                    : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white'
                )}
              >
                <item.icon className="h-5 w-5" />
                {item.name}
              </Link>
            )
          })}
        </div>

        {/* Cluster-Specific Navigation */}
        {isClusterSelected && (
          <div>
            <div className="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider px-3 pb-2">
              {selectedCluster}
            </div>
            {clusterNavigation.map((item) => {
              const href = item.href === '' 
                ? `/clusters/${selectedCluster}` 
                : `/clusters/${selectedCluster}${item.href}`
              // Use exact match for Dashboard, prefix match for sub-routes
              const isActive = item.href === '' 
                ? pathname === href 
                : pathname.startsWith(href)
              return (
                <Link
                  key={item.name}
                  href={href}
                  className={cn(
                    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white'
                      : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white'
                  )}
                >
                  <item.icon className="h-5 w-5" />
                  {item.name}
                </Link>
              )
            })}
          </div>
        )}
        
        {!isClusterSelected && (
          <div className="flex items-center justify-center py-8">
            <div className="text-center">
              <Cloud className="h-12 w-12 mx-auto text-gray-400 dark:text-gray-500 mb-2" />
              <p className="text-sm text-gray-500 dark:text-gray-400">Select a cluster to access</p>
              <p className="text-sm text-gray-500 dark:text-gray-400">models, tools, and agents</p>
            </div>
          </div>
        )}
      </nav>
      
      <div className="border-t p-3">
        <Link
          href="/settings"
          className={cn(
            'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
            pathname.startsWith('/settings')
              ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white'
              : 'text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-gray-900 dark:hover:text-white'
          )}
        >
          <Settings className="h-5 w-5" />
          Settings
        </Link>
      </div>
    </div>
  )
}
