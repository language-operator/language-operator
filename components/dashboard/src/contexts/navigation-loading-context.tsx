'use client'

import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useRouter, usePathname } from 'next/navigation'

interface NavigationLoadingContextType {
  isNavigating: boolean
  setNavigating: (loading: boolean) => void
  setReady: () => void
}

const NavigationLoadingContext = createContext<NavigationLoadingContextType | undefined>(undefined)

export function useNavigationLoading() {
  const context = useContext(NavigationLoadingContext)
  if (context === undefined) {
    throw new Error('useNavigationLoading must be used within a NavigationLoadingProvider')
  }
  return context
}

interface NavigationLoadingProviderProps {
  children: React.ReactNode
}

export function NavigationLoadingProvider({ children }: NavigationLoadingProviderProps) {
  const [isNavigating, setIsNavigating] = useState(false)
  const router = useRouter()
  const pathname = usePathname()

  const setNavigating = useCallback((loading: boolean) => {
    setIsNavigating(loading)
  }, [])

  const setReady = useCallback(() => {
    setIsNavigating(false)
  }, [])

  // Handle navigation events using browser APIs and Next.js events
  useEffect(() => {
    const handleStart = () => {
      console.log('%cNavigation started!', 'color: #22c55e; font-weight: bold;')
      setIsNavigating(true)
    }

    const handleComplete = () => {
      console.log('%cNavigation completed!', 'color: #f59e0b; font-weight: bold;')
      // Small delay to ensure page has rendered
      setTimeout(() => setIsNavigating(false), 100)
    }

    // Listen for click events on links
    const handleLinkClick = (e: Event) => {
      const target = e.target as HTMLElement
      const link = target.closest('a')
      
      if (link && link.href && link.href.startsWith(window.location.origin)) {
        // Only trigger for internal navigation
        if (link.href !== window.location.href) {
          handleStart()
        }
      }
    }

    // Listen for browser navigation (back/forward buttons)
    const handlePopState = () => {
      handleStart()
    }

    document.addEventListener('click', handleLinkClick)
    window.addEventListener('popstate', handlePopState)

    return () => {
      document.removeEventListener('click', handleLinkClick)
      window.removeEventListener('popstate', handlePopState)
    }
  }, [setIsNavigating])

  // Handle direct URL changes (browser navigation, link clicks)
  useEffect(() => {
    // Small delay to allow for component initialization
    const timer = setTimeout(() => {
      setIsNavigating(false)
    }, 100)

    return () => clearTimeout(timer)
  }, [pathname])

  const contextValue = {
    isNavigating,
    setNavigating,
    setReady,
  }

  return (
    <NavigationLoadingContext.Provider value={contextValue}>
      {children}
    </NavigationLoadingContext.Provider>
  )
}