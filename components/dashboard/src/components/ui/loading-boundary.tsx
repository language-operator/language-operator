'use client'

import React, { Suspense } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { AlertCircle, RefreshCw, Loader2 } from 'lucide-react'

interface LoadingBoundaryProps {
  children: React.ReactNode
  fallback?: React.ReactNode
  errorFallback?: (error: Error, retry: () => void) => React.ReactNode
  timeout?: number // milliseconds before showing timeout warning
  onTimeout?: () => void
}

interface LoadingState {
  isLoading: boolean
  error: Error | null
  hasTimedOut: boolean
  startTime: number
}

class LoadingBoundaryClass extends React.Component<
  LoadingBoundaryProps,
  LoadingState & { retryKey: number }
> {
  private timeoutId: NodeJS.Timeout | null = null

  constructor(props: LoadingBoundaryProps) {
    super(props)
    this.state = {
      isLoading: true,
      error: null,
      hasTimedOut: false,
      startTime: Date.now(),
      retryKey: 0
    }
  }

  componentDidMount() {
    this.startTimeout()
  }

  componentDidUpdate(prevProps: LoadingBoundaryProps, prevState: LoadingState & { retryKey: number }) {
    if (prevState.retryKey !== this.state.retryKey) {
      this.resetState()
      this.startTimeout()
    }
  }

  componentWillUnmount() {
    this.clearTimeout()
  }

  startTimeout = () => {
    this.clearTimeout()
    const timeout = this.props.timeout || 15000 // 15 seconds default
    
    this.timeoutId = setTimeout(() => {
      if (this.state.isLoading && !this.state.error) {
        this.setState({ hasTimedOut: true })
        this.props.onTimeout?.()
      }
    }, timeout)
  }

  clearTimeout = () => {
    if (this.timeoutId) {
      clearTimeout(this.timeoutId)
      this.timeoutId = null
    }
  }

  resetState = () => {
    this.setState({
      isLoading: true,
      error: null,
      hasTimedOut: false,
      startTime: Date.now()
    })
  }

  retry = () => {
    this.setState(prevState => ({
      retryKey: prevState.retryKey + 1
    }))
  }

  static getDerivedStateFromError(error: Error) {
    return {
      error,
      isLoading: false,
      hasTimedOut: false
    }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('LoadingBoundary caught an error:', error, errorInfo)
    this.clearTimeout()
  }

  render() {
    const { children, fallback, errorFallback } = this.props
    const { error, hasTimedOut } = this.state

    // Error state
    if (error) {
      if (errorFallback) {
        return errorFallback(error, this.retry)
      }

      return (
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-red-700">
              <AlertCircle className="h-5 w-5" />
              Loading Error
            </CardTitle>
            <CardDescription className="text-red-600">
              Failed to load content
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  {error.message || 'An unexpected error occurred'}
                </AlertDescription>
              </Alert>
              <Button onClick={this.retry} variant="outline" className="w-full">
                <RefreshCw className="h-4 w-4 mr-2" />
                Try Again
              </Button>
            </div>
          </CardContent>
        </Card>
      )
    }

    // Timeout warning state
    if (hasTimedOut) {
      return (
        <Card className="border-yellow-200 bg-yellow-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-yellow-700">
              <Loader2 className="h-5 w-5 animate-spin" />
              Still Loading...
            </CardTitle>
            <CardDescription className="text-yellow-600">
              This is taking longer than expected
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <Alert className="border-yellow-200 bg-yellow-50">
                <AlertCircle className="h-4 w-4 text-yellow-600" />
                <AlertDescription className="text-yellow-700">
                  The page is still loading. You can wait a bit longer or try refreshing.
                </AlertDescription>
              </Alert>
              <div className="flex gap-2">
                <Button onClick={this.retry} variant="outline" className="flex-1">
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Refresh
                </Button>
                <Button 
                  onClick={() => window.location.reload()} 
                  variant="outline" 
                  className="flex-1"
                >
                  Reload Page
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )
    }

    // Success state
    return (
      <React.Fragment key={this.state.retryKey}>
        <Suspense fallback={fallback || <LoadingSkeleton />}>
          {children}
        </Suspense>
      </React.Fragment>
    )
  }
}

/**
 * Default loading skeleton component
 */
export function LoadingSkeleton() {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center space-x-2">
          <Skeleton className="h-5 w-5 rounded" />
          <Skeleton className="h-5 w-32" />
        </div>
        <Skeleton className="h-4 w-64" />
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Resource loading skeleton for dashboard cards
 */
export function ResourceLoadingSkeleton() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
      {[...Array(4)].map((_, i) => (
        <Card key={i}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-4" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-8 mb-2" />
            <Skeleton className="h-3 w-20" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

/**
 * Cluster dashboard loading skeleton
 */
export function ClusterDashboardSkeleton() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Skeleton className="h-8 w-8 rounded" />
          <div>
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-4 w-32 mt-1" />
          </div>
        </div>
        <Skeleton className="h-9 w-20" />
      </div>

      {/* Info cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-48" />
          </CardHeader>
          <CardContent className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i}>
                <Skeleton className="h-4 w-16 mb-1" />
                <Skeleton className="h-4 w-24" />
              </div>
            ))}
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-28" />
            <Skeleton className="h-4 w-40" />
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="flex flex-col items-center p-4 border rounded-lg">
                  <Skeleton className="h-8 w-8 mb-2" />
                  <Skeleton className="h-4 w-20 mb-1" />
                  <Skeleton className="h-3 w-16" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Resource cards */}
      <ResourceLoadingSkeleton />
    </div>
  )
}

export default LoadingBoundaryClass
export { LoadingBoundaryClass as LoadingBoundary }