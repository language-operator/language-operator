'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { cn } from '@/lib/utils'
import { useClusterContext } from '@/contexts/cluster-context'
import { ClusterSelector } from '@/components/cluster-selector'
import { useSidebarContext } from '@/contexts/sidebar-context'
import { useOrganization } from '@/components/organization-provider'
import {
  Home,
  Bot,
  Cpu,
  Wrench,
  Users,
  Boxes,
  Settings,
  BarChart3,
  MessageSquare,
  Palette,
  ChevronLeft,
  Gauge,
  Server,
} from 'lucide-react'

const globalNavigation = [
  { name: 'Overview', href: '/', icon: Home },
  { name: 'Capacity', href: '/capacity', icon: Server },
  { name: 'Clusters', href: '/clusters', icon: Boxes },
]

const clusterNavigation = [
  { name: 'Dashboard', href: '', icon: Gauge }, // Will be prefixed with /clusters/[name]
  { name: 'Console', href: '/console', icon: MessageSquare },
  { name: 'Agents', href: '/agents', icon: Bot },
  { name: 'Tools', href: '/tools', icon: Wrench },
  { name: 'Personas', href: '/personas', icon: Users },
  { name: 'Models', href: '/models', icon: Cpu },
]

export function Sidebar() {
  const pathname = usePathname()
  const { selectedCluster, isClusterSelected } = useClusterContext()
  const { isCollapsed, setIsCollapsed, isLoaded } = useSidebarContext()
  
  // Try to get organization context, but don't crash if not available (e.g., settings pages)
  const orgContext = (() => {
    try {
      return useOrganization()
    } catch {
      return null
    }
  })()
  
  const getOrgUrl = orgContext?.getOrgUrl || ((path: string) => path)
  
  // Check if we're on a console page to remove background overrides
  const isConsolePage = pathname.includes('/console')

  // Prevent hydration mismatch by not rendering until localStorage is loaded
  if (!isLoaded) {
    return (
      <div className="flex h-screen w-64 flex-col bg-stone-100 border-r border-stone-800/80 dark:bg-stone-950 dark:border-stone-600/80">
        {/* Skeleton loader while loading state */}
      </div>
    )
  }

  return (
    <div className={cn(
      "flex h-screen flex-col border-r border-stone-800/80 dark:border-stone-600/80 transition-all duration-300",
      isCollapsed ? "w-16" : "w-64",
      !isConsolePage && "bg-stone-100 dark:bg-stone-950"
    )}>
      <div className="flex h-16 items-center border-b border-stone-800/80 px-4 dark:border-stone-600/80">
        <div className="w-full">
          {!isCollapsed ? (
            <h1 className="text-[13px] font-light tracking-widest uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
              Language Operator
              <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
            </h1>
          ) : (
            <div className="flex items-center justify-center">
              <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
            </div>
          )}
        </div>
      </div>
      
      {/* Cluster Selector */}
      {!isCollapsed && <ClusterSelector />}
      
      <nav className={cn(
        "flex-1 space-y-1",
        isCollapsed ? "px-2 py-2" : "px-4 py-6"
      )}>
        {/* Global Navigation */}
        <div className="mb-6">
          {!isCollapsed && (
            <div className="text-[10px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400 px-3 pb-2">
              Global
            </div>
          )}
          {globalNavigation.map((item) => {
            const href = getOrgUrl(item.href)
            const isActive = pathname === href
            return (
              <Link
                key={item.name}
                href={href}
                className={cn(
                  'flex items-center transition-colors',
                  isCollapsed 
                    ? 'justify-center p-3 mx-auto my-1 rounded' 
                    : 'gap-3 px-3 py-3 border-l-2',
                  isActive
                    ? isCollapsed
                      ? 'text-amber-400 dark:text-amber-400'
                      : 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
                    : isCollapsed
                      ? 'text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500'
                      : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
                )}
                title={isCollapsed ? item.name : undefined}
              >
                <item.icon className="h-5 w-5 flex-shrink-0" />
                {!isCollapsed && <span className="text-sm font-light">{item.name}</span>}
              </Link>
            )
          })}
        </div>

        {/* Cluster-Specific Navigation */}
        {isClusterSelected && (
          <div>
            {!isCollapsed && (
              <div className="text-[10px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400 px-3 pb-2">
                {selectedCluster}
              </div>
            )}
            {clusterNavigation.map((item) => {
              const clusterPath = item.href === '' 
                ? `/clusters/${selectedCluster}` 
                : `/clusters/${selectedCluster}${item.href}`
              const href = getOrgUrl(clusterPath)
              // Use exact match for Dashboard, prefix match for sub-routes
              const isActive = item.href === '' 
                ? pathname === href 
                : pathname.startsWith(href)
              return (
                <Link
                  key={item.name}
                  href={href}
                  className={cn(
                    'flex items-center transition-colors',
                    isCollapsed 
                      ? 'justify-center p-3 mx-auto my-1 rounded' 
                      : 'gap-3 px-3 py-3 border-l-2',
                    isActive
                      ? isCollapsed
                        ? 'text-amber-400 dark:text-amber-400'
                        : 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
                      : isCollapsed
                        ? 'text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500'
                        : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
                  )}
                  title={isCollapsed ? item.name : undefined}
                >
                  <item.icon className="h-5 w-5 flex-shrink-0" />
                  {!isCollapsed && <span className="text-sm font-light">{item.name}</span>}
                </Link>
              )
            })}
          </div>
        )}
        
        {!isClusterSelected && !isCollapsed && (
          <div className="flex items-center justify-center py-8">
            <div className="text-center">
              <Boxes className="h-12 w-12 mx-auto text-stone-400 dark:text-stone-500 mb-2" />
              <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">Select a cluster to access</p>
              <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">models, tools, and agents</p>
            </div>
          </div>
        )}
      </nav>
      
      <div className={cn(
        "border-t border-stone-800/80 dark:border-stone-600/80 space-y-1",
        isCollapsed ? "p-2" : "p-4"
      )}>
        <Link
          href={getOrgUrl('/settings/users')}
          className={cn(
            'flex items-center transition-colors',
            isCollapsed 
              ? 'justify-center p-3 mx-auto my-1 rounded' 
              : 'gap-3 px-3 py-3 border-l-2',
            pathname.startsWith('/settings')
              ? isCollapsed
                ? 'text-amber-400 dark:text-amber-400'
                : 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
              : isCollapsed
                ? 'text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500'
                : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
          )}
          title={isCollapsed ? 'Settings' : undefined}
        >
          <Settings className="h-5 w-5 flex-shrink-0" />
          {!isCollapsed && <span className="text-sm font-light">Settings</span>}
        </Link>
        
        <Link
          href="/styleguide"
          target="_blank"
          rel="noopener noreferrer"
          className={cn(
            'flex items-center transition-colors',
            isCollapsed 
              ? 'justify-center p-3 mx-auto my-1 rounded' 
              : 'gap-3 px-3 py-3 border-l-2',
            pathname === '/styleguide'
              ? isCollapsed
                ? 'text-amber-400 dark:text-amber-400'
                : 'bg-stone-100 text-stone-900 border-stone-900 dark:bg-stone-800 dark:text-stone-300 dark:border-amber-400'
              : isCollapsed
                ? 'text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500'
                : 'text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
          )}
          title={isCollapsed ? 'Style Guide' : undefined}
        >
          <Palette className="h-5 w-5 flex-shrink-0" />
          {!isCollapsed && <span className="text-sm font-light">Style Guide</span>}
        </Link>
        
        {/* Collapse Toggle */}
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          className={cn(
            'flex items-center transition-all duration-300',
            isCollapsed 
              ? 'justify-center p-3 mx-auto my-1 rounded text-stone-600 hover:bg-stone-100 hover:text-amber-900 dark:text-stone-400 dark:hover:bg-stone-800 dark:hover:text-amber-500'
              : 'gap-3 px-3 py-3 border-l-2 text-stone-600 border-transparent hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500'
          )}
          title={isCollapsed ? 'Expand Sidebar' : 'Collapse Sidebar'}
        >
          <ChevronLeft className={cn(
            "h-5 w-5 transition-transform duration-300 flex-shrink-0",
            isCollapsed && "rotate-180"
          )} />
          {!isCollapsed && <span className="text-sm font-light">Collapse</span>}
        </button>
      </div>
    </div>
  )
}
