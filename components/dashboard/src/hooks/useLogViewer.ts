import { useState, useEffect, useRef, useCallback } from 'react'

export interface UseLogViewerOptions {
  autoScroll?: boolean
  maxLines?: number
}

export interface UseLogViewerReturn {
  // State
  logs: string[]
  loading: boolean
  error: string | null
  isAtBottom: boolean
  userScrolledUp: boolean
  
  // Refs
  logsEndRef: React.RefObject<HTMLDivElement | null>
  logsContainerRef: React.RefObject<HTMLDivElement | null>
  
  // Actions
  setLogs: React.Dispatch<React.SetStateAction<string[]>>
  setLoading: React.Dispatch<React.SetStateAction<boolean>>
  setError: React.Dispatch<React.SetStateAction<string | null>>
  scrollToBottom: () => void
  clearLogs: () => void
  handleScroll: () => void
}

export function useLogViewer(options: UseLogViewerOptions = {}): UseLogViewerReturn {
  const { autoScroll = true } = options
  
  // State
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [userScrolledUp, setUserScrolledUp] = useState(false)
  
  // Refs
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = useCallback(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    setIsAtBottom(true)
    setUserScrolledUp(false)
  }, [])

  const checkScrollPosition = useCallback(() => {
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
      const isNearBottom = scrollTop + clientHeight >= scrollHeight - 10
      setIsAtBottom(isNearBottom)
    }
  }, [])

  const handleScroll = useCallback(() => {
    checkScrollPosition()
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
      const isNearBottom = scrollTop + clientHeight >= scrollHeight - 10
      if (!isNearBottom) {
        setUserScrolledUp(true)
      }
    }
  }, [checkScrollPosition])

  const clearLogs = useCallback(() => {
    setLogs([])
    setIsAtBottom(true)
    setUserScrolledUp(false)
  }, [])

  // Auto-scroll effect - only if user is at bottom and hasn't scrolled up
  useEffect(() => {
    if (autoScroll && isAtBottom && !userScrolledUp) {
      scrollToBottom()
    }
  }, [logs, isAtBottom, userScrolledUp, autoScroll])

  // Check scroll position after logs are loaded
  useEffect(() => {
    const timer = setTimeout(() => {
      checkScrollPosition()
    }, 100)
    return () => clearTimeout(timer)
  }, [logs])

  return {
    // State
    logs,
    loading,
    error,
    isAtBottom,
    userScrolledUp,
    
    // Refs
    logsEndRef,
    logsContainerRef,
    
    // Actions
    setLogs,
    setLoading,
    setError,
    scrollToBottom,
    clearLogs,
    handleScroll,
  }
}