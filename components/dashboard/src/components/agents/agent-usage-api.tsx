'use client'

import { useState, useMemo, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CalendarIcon, BarChart3, DollarSign, Clock, Zap, AlertTriangle, Target, TrendingUp, ChevronLeft, ChevronRight } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'
import { AdvancedDateRangePicker } from '@/components/ui/advanced-date-range-picker'
import { UsageCostChart } from '@/components/agents/usage-cost-chart'
import { TaskCostBreakdownChart } from '@/components/agents/task-cost-breakdown-chart'
import { Sparkline } from '@/components/ui/sparkline'
import { formatCurrencyAutoPrecision, getUserLocale } from '@/lib/currency'

interface AgentUsageProps {
  agent: LanguageAgent
  clusterName: string
}

// API Response Types
interface UsageDataPoint {
  timestamp: string
  cost: number
  tokens: number
  tasks: number
  avgResponseTime: number
  errors: number
}

interface TaskCostDataPoint {
  timestamp: string
  [taskName: string]: string | number
}

interface UsageMetrics {
  totalCost: number
  totalTokens: number
  totalTasks: number
  totalErrors: number
  avgResponseTime: number
  costPerTask: number
  errorRate: number
}

interface UsageApiResponse {
  metrics: UsageMetrics
  timeSeries: UsageDataPoint[]
  taskBreakdown: {
    data: TaskCostDataPoint[]
    taskNames: string[]
  }
  period: {
    from: string
    to: string
    granularity: string
  }
  currency?: string
}

// Convert API data to chart format
function transformTimeSeriesData(timeSeries: UsageDataPoint[]) {
  return timeSeries.map(point => ({
    date: point.timestamp, // Keep full timestamp for hour-level granularity
    cost: point.cost,
    tokens: point.tokens,
    tasks: point.tasks,
    avgResponseTime: point.avgResponseTime,
    errors: point.errors
  }))
}

export function AgentUsage({ agent, clusterName }: AgentUsageProps) {
  const router = useRouter()
  const searchParams = useSearchParams()
  
  // Initialize date range from URL params or default to last 7 days
  const [dateRange, setDateRange] = useState<{ from: Date; to: Date }>(() => {
    const fromParam = searchParams.get('from')
    const toParam = searchParams.get('to')
    
    if (fromParam && toParam) {
      return {
        from: new Date(fromParam),
        to: new Date(toParam)
      }
    }
    
    return {
      from: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000), // 2 days ago
      to: new Date()
    }
  })
  const [isDatePickerOpen, setIsDatePickerOpen] = useState(false)
  
  // Update URL when date range changes
  const handleDateRangeChange = (newRange: { from: Date; to: Date }) => {
    setDateRange(newRange)
    
    // Update URL search params
    const params = new URLSearchParams(searchParams.toString())
    params.set('from', newRange.from.toISOString())
    params.set('to', newRange.to.toISOString())
    
    router.push(`?${params.toString()}`, { scroll: false })
  }
  
  // Calculate period duration and navigation
  const periodDurationMs = dateRange.to.getTime() - dateRange.from.getTime()
  
  const goToPreviousPeriod = () => {
    const newFrom = new Date(dateRange.from.getTime() - periodDurationMs)
    const newTo = new Date(dateRange.to.getTime() - periodDurationMs)
    handleDateRangeChange({ from: newFrom, to: newTo })
  }
  
  const goToNextPeriod = () => {
    const newFrom = new Date(dateRange.from.getTime() + periodDurationMs)
    const newTo = new Date(dateRange.to.getTime() + periodDurationMs)
    handleDateRangeChange({ from: newFrom, to: newTo })
  }
  
  // Check if next period would be in the future
  const now = new Date()
  const nextPeriodStart = new Date(dateRange.from.getTime() + periodDurationMs)
  const isNextPeriodDisabled = nextPeriodStart > now

  // Fetch usage data from API
  const { data: usageData, isLoading, error } = useQuery<UsageApiResponse>({
    queryKey: ['cost', clusterName, agent.metadata.name, dateRange.from.toISOString(), dateRange.to.toISOString()],
    queryFn: async () => {
      const params = new URLSearchParams({
        from: dateRange.from.toISOString(),
        to: dateRange.to.toISOString(),
        // granularity will be automatically determined by the API based on date range
      })
      
      const response = await fetch(`/api/clusters/${clusterName}/agents/${agent.metadata.name}/cost?${params}`)
      
      if (!response.ok) {
        throw new Error(`Failed to fetch cost data: ${response.statusText}`)
      }
      
      return response.json()
    },
    staleTime: 60 * 1000, // Cache for 1 minute
    refetchOnWindowFocus: false,
  })

  // Transform API data for charts
  const chartData = useMemo(() => {
    if (!usageData) return []
    return transformTimeSeriesData(usageData.timeSeries)
  }, [usageData])

  // Calculate sparkline data and derived metrics from API data
  const metrics = useMemo(() => {
    if (!usageData) {
      return {
        totalCost: formatCurrencyAutoPrecision(0),
        totalTokens: '0',
        totalTasks: '0',
        avgResponseTime: '0.0',
        errorRate: '0.0',
        costPerTask: formatCurrencyAutoPrecision(0),
        tokensPerMinute: '0',
        costPerDay: formatCurrencyAutoPrecision(0),
        costPerMonth: formatCurrencyAutoPrecision(0),
        avgDailyTokens: '0',
        sparklines: {
          cost: [],
          tokens: [],
          tasks: [],
          responseTime: [],
          errors: []
        }
      }
    }

    const { metrics: apiMetrics, timeSeries, currency } = usageData
    const days = timeSeries?.length || 1
    
    // Helper function to calculate rolling average
    const calculateRollingAverage = (data: number[], windowSize: number = 3): number[] => {
      if (data.length <= windowSize) return data
      
      return data.map((_, index) => {
        const start = Math.max(0, index - Math.floor(windowSize / 2))
        const end = Math.min(data.length - 1, index + Math.floor(windowSize / 2))
        const window = data.slice(start, end + 1)
        return window.reduce((sum, val) => sum + val, 0) / window.length
      })
    }
    
    // Generate sparkline data arrays with consistent sampling across the selected period
    // Since timeSeries is already padded with complete intervals, sample evenly for sparklines
    const maxSparklinePoints = 15
    const totalPoints = timeSeries?.length || 0
    
    let sparklineIndices: number[] = []
    if (totalPoints <= maxSparklinePoints) {
      // Use all points if we have few enough
      sparklineIndices = Array.from({ length: totalPoints }, (_, i) => i)
    } else {
      // Sample evenly across the entire time range
      const step = (totalPoints - 1) / (maxSparklinePoints - 1)
      sparklineIndices = Array.from({ length: maxSparklinePoints }, (_, i) => 
        Math.round(i * step)
      )
    }
    
    const rawSparklineData = {
      cost: sparklineIndices.map(i => timeSeries?.[i]?.cost || 0),
      tokens: sparklineIndices.map(i => timeSeries?.[i]?.tokens || 0),
      tasks: sparklineIndices.map(i => timeSeries?.[i]?.tasks || 0),
      responseTime: sparklineIndices.map(i => timeSeries?.[i]?.avgResponseTime || 0),
      errors: sparklineIndices.map(i => timeSeries?.[i]?.errors || 0)
    }

    const sparklineData = {
      cost: rawSparklineData.cost,
      costAverage: calculateRollingAverage(rawSparklineData.cost, 7), // 7-day rolling average for "Avg Daily Cost"
      tokens: rawSparklineData.tokens, // Raw token values for "Tokens Spent" 
      tokensAverage: calculateRollingAverage(rawSparklineData.tokens, 7), // 7-day rolling average for "Avg Daily Tokens"
      tasks: rawSparklineData.tasks,
      responseTime: rawSparklineData.responseTime,
      errors: rawSparklineData.errors
    }

    return {
      totalCost: formatCurrencyAutoPrecision(apiMetrics?.totalCost || 0, currency),
      totalTokens: (apiMetrics?.totalTokens || 0).toLocaleString(),
      totalTasks: (apiMetrics?.totalTasks || 0).toLocaleString(),
      avgResponseTime: (apiMetrics?.avgResponseTime || 0).toFixed(1),
      errorRate: (apiMetrics?.errorRate || 0).toFixed(1),
      costPerTask: formatCurrencyAutoPrecision(apiMetrics?.costPerTask || 0, currency),
      tokensPerMinute: Math.round((apiMetrics?.totalTokens || 0) / (days * 24 * 60)).toLocaleString(),
      costPerDay: formatCurrencyAutoPrecision((apiMetrics?.totalCost || 0) / days, currency),
      costPerMonth: formatCurrencyAutoPrecision((apiMetrics?.totalCost || 0) / days * 30, currency),
      avgDailyTokens: Math.round((apiMetrics?.totalTokens || 0) / days).toLocaleString(),
      sparklines: sparklineData
    }
  }, [usageData])

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent>
            <div className="flex items-center gap-3">
              <Button variant="outline" size="icon" disabled className="h-12 w-12 shrink-0">
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div className="flex-1">
                <div className="h-12 bg-muted animate-pulse rounded-md" />
              </div>
              <Button variant="outline" size="icon" disabled className="h-12 w-12 shrink-0">
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-6">
                <div className="space-y-2">
                  <div className="h-4 bg-muted animate-pulse rounded" />
                  <div className="h-8 bg-muted animate-pulse rounded" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="p-6">
            <div className="text-center space-y-2">
              <AlertTriangle className="h-8 w-8 text-destructive mx-auto" />
              <h3 className="font-medium">Failed to load usage data</h3>
              <p className="text-sm text-muted-foreground">
                {error instanceof Error ? error.message : 'Unknown error occurred'}
              </p>
              <Button variant="outline" onClick={() => window.location.reload()}>
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Date Range Picker */}
      <Card>
        <CardContent>
          <div className="flex items-center gap-3">
            {/* Previous Period Button */}
            <Button
              variant="outline"
              size="icon"
              onClick={goToPreviousPeriod}
              className="h-12 w-12 shrink-0"
              title="Previous period"
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            
            {/* Date Range Picker */}
            <div className="flex-1">
              <AdvancedDateRangePicker
                date={dateRange}
                onDateChange={handleDateRangeChange}
                onOpenChange={setIsDatePickerOpen}
              />
            </div>
            
            {/* Next Period Button */}
            <Button
              variant="outline"
              size="icon"
              onClick={goToNextPeriod}
              disabled={isNextPeriodDisabled}
              className="h-12 w-12 shrink-0"
              title="Next period"
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Total Cost */}
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 opacity-25">
            <Sparkline 
              data={metrics.sparklines.cost} 
              color="#3b82f6" 
              className="w-full h-full"
            />
          </div>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="relative z-10">
            <div className="text-2xl font-bold">{metrics.totalCost}</div>
            <p className="text-xs text-muted-foreground">
              Selected period
            </p>
          </CardContent>
        </Card>

        {/* Avg Daily Cost */}
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 opacity-25">
            <Sparkline 
              data={metrics.sparklines.cost}
              color="#3b82f6" 
              className="w-full h-full"
            />
          </div>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
            <CardTitle className="text-sm font-medium">Avg Daily Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="relative z-10">
            <div className="text-2xl font-bold">{metrics.costPerDay}</div>
            <p className="text-xs text-muted-foreground">
              Per day
            </p>
          </CardContent>
        </Card>

        {/* Tokens Spent */}
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 opacity-25">
            <Sparkline 
              data={metrics.sparklines.tokens} 
              color="#3b82f6" 
              className="w-full h-full"
            />
          </div>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
            <CardTitle className="text-sm font-medium">Tokens Spent</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="relative z-10">
            <div className="text-2xl font-bold">{metrics.totalTokens}</div>
            <p className="text-xs text-muted-foreground">
              Selected period
            </p>
          </CardContent>
        </Card>

        {/* Avg Daily Tokens */}
        <Card className="relative overflow-hidden">
          <div className="absolute inset-0 opacity-25">
            <Sparkline 
              data={metrics.sparklines.tokens}
              color="#3b82f6" 
              className="w-full h-full"
            />
          </div>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 relative z-10">
            <CardTitle className="text-sm font-medium">Avg Daily Tokens</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent className="relative z-10">
            <div className="text-2xl font-bold">{metrics.avgDailyTokens}</div>
            <p className="text-xs text-muted-foreground">
              Per day
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Cost Over Time Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Cost Over Time
          </CardTitle>
          <CardDescription>
            Daily cost breakdown for agent operations
          </CardDescription>
        </CardHeader>
        <CardContent>
          <UsageCostChart 
            data={chartData} 
            granularity={usageData?.period.granularity || 'day'} 
            currency={usageData?.currency}
          />
        </CardContent>
      </Card>

      {/* Task Cost Breakdown Chart */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Task Cost Over Time
          </CardTitle>
          <CardDescription>
            Cost breakdown by individual tasks over time
          </CardDescription>
        </CardHeader>
        <CardContent>
          <TaskCostBreakdownChart 
            data={usageData?.taskBreakdown.data?.map(item => ({
              ...item,
              date: item.timestamp
            })) || []} 
            taskNames={usageData?.taskBreakdown.taskNames || []} 
            granularity={usageData?.period.granularity || 'day'}
            currency={usageData?.currency}
          />
        </CardContent>
      </Card>
    </div>
  )
}