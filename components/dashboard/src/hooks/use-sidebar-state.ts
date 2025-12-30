import { useState, useEffect } from 'react'

const SIDEBAR_STORAGE_KEY = 'sidebar-collapsed'

export function useSidebarState() {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [isLoaded, setIsLoaded] = useState(false)

  // Load initial state from localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return

    try {
      const stored = localStorage.getItem(SIDEBAR_STORAGE_KEY)
      if (stored !== null) {
        setIsCollapsed(JSON.parse(stored))
      }
    } catch (error) {
      console.error('Failed to load sidebar state:', error)
    } finally {
      setIsLoaded(true)
    }
  }, [])

  // Save state to localStorage whenever it changes
  useEffect(() => {
    if (!isLoaded) return
    if (typeof window === 'undefined') return

    try {
      localStorage.setItem(SIDEBAR_STORAGE_KEY, JSON.stringify(isCollapsed))
    } catch (error) {
      console.error('Failed to save sidebar state:', error)
    }
  }, [isCollapsed, isLoaded])

  return {
    isCollapsed,
    setIsCollapsed,
    isLoaded
  }
}