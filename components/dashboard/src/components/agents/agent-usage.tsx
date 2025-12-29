'use client'

import { useState, useMemo, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CalendarIcon, BarChart3, DollarSign, Clock, Zap, AlertTriangle, Target, TrendingUp, ChevronLeft, ChevronRight } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'
import { AdvancedDateRangePicker } from '@/components/ui/advanced-date-range-picker'
import { UsageCostChart } from '@/components/agents/usage-cost-chart'
import { TaskCostBreakdownChart } from '@/components/agents/task-cost-breakdown-chart'
import { Sparkline } from '@/components/ui/sparkline'

interface AgentUsageProps {
  agent: LanguageAgent
  clusterName: string
}

// Mock data for development
const generateMockUsageData = (days: number) => {
  const data = []
  const startDate = new Date()
  startDate.setDate(startDate.getDate() - days)
  
  for (let i = 0; i < days; i++) {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + i)
    
    // Generate realistic patterns
    const isWeekday = date.getDay() >= 1 && date.getDay() <= 5
    const baseUsage = isWeekday ? 0.8 : 0.3
    const randomFactor = Math.random() * 0.6 + 0.7 // 0.7 to 1.3
    
    data.push({
      date: date.toISOString().split('T')[0],
      cost: parseFloat((baseUsage * randomFactor * 2.5).toFixed(2)),
      tokens: Math.floor(baseUsage * randomFactor * 50000),
      tasks: Math.floor(baseUsage * randomFactor * 25),
      avgResponseTime: parseFloat((1.2 + Math.random() * 0.8).toFixed(1)),
      errors: Math.floor(Math.random() * 3)
    })
  }
  
  return data
}

// Mock task cost breakdown data
const generateMockTaskCostData = (days: number) => {
  const taskNames = ['read_story_file', 'generate_plot_modification', 'update_story_file', 'read_memory_file', 'update_memory_file']
  const data = []
  const startDate = new Date()
  startDate.setDate(startDate.getDate() - days)
  
  for (let i = 0; i < days; i++) {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + i)
    
    // Generate realistic patterns - some tasks cost more than others
    const isWeekday = date.getDay() >= 1 && date.getDay() <= 5
    const baseMultiplier = isWeekday ? 1.0 : 0.4
    
    const dayData: any = {
      date: date.toISOString().split('T')[0],
    }
    
    // Generate cost for each task with different cost profiles
    taskNames.forEach((taskName, index) => {
      let taskBaseCost = 0.1 // Default base cost
      
      // Different tasks have different cost profiles
      switch (taskName) {
        case 'generate_plot_modification':
          taskBaseCost = 1.2 // Most expensive (AI generation)
          break
        case 'read_story_file':
        case 'read_memory_file':
          taskBaseCost = 0.05 // Cheap reads
          break
        case 'update_story_file':
        case 'update_memory_file':
          taskBaseCost = 0.2 // Moderate writes
          break
      }
      
      const randomFactor = Math.random() * 0.6 + 0.7 // 0.7 to 1.3
      dayData[taskName] = parseFloat((taskBaseCost * baseMultiplier * randomFactor).toFixed(3))
    })
    
    data.push(dayData)
  }
  
  return { data, taskNames }
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
      from: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // 7 days ago
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

  // Generate mock data based on date range
  const mockData = useMemo(() => {
    const days = Math.ceil((dateRange.to.getTime() - dateRange.from.getTime()) / (24 * 60 * 60 * 1000))
    return generateMockUsageData(Math.min(days, 90)) // Cap at 90 days for performance
  }, [dateRange])

  // Generate mock task cost breakdown data
  const taskCostData = useMemo(() => {
    const days = Math.ceil((dateRange.to.getTime() - dateRange.from.getTime()) / (24 * 60 * 60 * 1000))
    return generateMockTaskCostData(Math.min(days, 90))
  }, [dateRange])

  // Calculate summary metrics and sparkline data from mock data
  const metrics = useMemo(() => {
    const totalCost = mockData.reduce((sum, day) => sum + day.cost, 0)
    const totalTokens = mockData.reduce((sum, day) => sum + day.tokens, 0)
    const totalTasks = mockData.reduce((sum, day) => sum + day.tasks, 0)
    const totalErrors = mockData.reduce((sum, day) => sum + day.errors, 0)
    const totalResponseTime = mockData.reduce((sum, day) => sum + day.avgResponseTime, 0)
    const days = mockData.length || 1
    
    const avgResponseTime = totalResponseTime / days
    const errorRate = totalTasks > 0 ? (totalErrors / totalTasks) * 100 : 0
    const costPerTask = totalTasks > 0 ? totalCost / totalTasks : 0
    const tokensPerMinute = totalTokens / (days * 24 * 60) // Assuming even distribution
    const costPerDay = totalCost / days
    const costPerMonth = costPerDay * 30
    const avgDailyTokens = totalTokens / days

    // Generate sparkline data arrays (last 15 data points for clean visualization)
    const recentData = mockData.slice(-15)
    const sparklineData = {
      cost: recentData.map(d => d.cost),
      tokens: recentData.map(d => d.tokens),
      tasks: recentData.map(d => d.tasks),
      responseTime: recentData.map(d => d.avgResponseTime),
      errors: recentData.map(d => d.errors),
      // Calculate derived sparklines
      tokensPerMinuteDaily: recentData.map(d => d.tokens / (24 * 60)),
      costPerTaskDaily: recentData.map(d => d.tasks > 0 ? d.cost / d.tasks : 0),
      errorRateDaily: recentData.map(d => d.tasks > 0 ? (d.errors / d.tasks) * 100 : 0)
    }

    return {
      totalCost: totalCost.toFixed(2),
      totalTokens: totalTokens.toLocaleString(),
      totalTasks: totalTasks.toLocaleString(),
      avgResponseTime: avgResponseTime.toFixed(1),
      errorRate: errorRate.toFixed(1),
      costPerTask: costPerTask.toFixed(3),
      tokensPerMinute: Math.round(tokensPerMinute).toLocaleString(),
      costPerDay: costPerDay.toFixed(2),
      costPerMonth: costPerMonth.toFixed(2),
      avgDailyTokens: Math.round(avgDailyTokens).toLocaleString(),
      sparklines: sparklineData
    }
  }, [mockData])

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
            <div className="text-2xl font-bold">${metrics.totalCost}</div>
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
            <div className="text-2xl font-bold">${metrics.costPerDay}</div>
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
          <UsageCostChart data={mockData} />
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
            data={taskCostData.data} 
            taskNames={taskCostData.taskNames} 
          />
        </CardContent>
      </Card>
    </div>
  )
}