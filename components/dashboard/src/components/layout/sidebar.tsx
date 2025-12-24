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
    <div className="flex h-screen w-64 flex-col bg-white border-r border-stone-800/80 dark:bg-stone-900 dark:border-stone-600/80">
      <div className="flex h-16 items-center border-b border-stone-800/80 px-4 dark:border-stone-600/80">
        <div className="w-full">
          <h1 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
            Language Operator
            <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
          </h1>
        </div>
      </div>
      
      {/* Cluster Selector */}
      <ClusterSelector />
      
      <nav className="flex-1 space-y-1 px-4 py-6">
        {/* Global Navigation */}
        <div className="mb-6">
          <div className="text-[10px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400 px-3 pb-2">
            Global
          </div>
          {globalNavigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 px-3 py-2 text-sm font-light transition-colors border-l-2',
                  isActive
                    ? 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
                    : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
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
            <div className="text-[10px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400 px-3 pb-2">
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
                    'flex items-center gap-3 px-3 py-2 text-sm font-light transition-colors border-l-2',
                    isActive
                      ? 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
                      : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
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
              <Cloud className="h-12 w-12 mx-auto text-stone-400 dark:text-stone-500 mb-2" />
              <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">Select a cluster to access</p>
              <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">models, tools, and agents</p>
            </div>
          </div>
        )}
      </nav>
      
      <div className="border-t border-stone-800/80 dark:border-stone-600/80 p-4 space-y-1">
        <Link
          href="/settings/users"
          className={cn(
            'flex items-center gap-3 px-3 py-2 text-sm font-light transition-colors border-l-2',
            pathname.startsWith('/settings')
              ? 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
              : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
          )}
        >
          <Settings className="h-5 w-5" />
          Settings
        </Link>
        
        <Link
          href="/styleguide"
          className={cn(
            'flex items-center gap-3 px-3 py-2 text-sm font-light transition-colors border-l-2',
            pathname === '/styleguide'
              ? 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
              : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
          )}
        >
          <div className="h-5 w-5 bg-stone-600 dark:bg-stone-400" />
          Style Guide
        </Link>
      </div>
    </div>
  )
}
