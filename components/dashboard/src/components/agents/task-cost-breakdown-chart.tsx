'use client'

import { useMemo, useState } from 'react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { useTheme } from 'next-themes'
import { formatCurrencyAutoPrecision } from '@/lib/currency'

interface TaskCostData {
  date: string
  [taskName: string]: string | number // Dynamic task names as keys with cost values
}

interface TaskCostBreakdownChartProps {
  data: TaskCostData[]
  taskNames: string[]
  granularity?: string
  currency?: string
}

export function TaskCostBreakdownChart({ data, taskNames, granularity = 'day', currency = 'USD' }: TaskCostBreakdownChartProps) {
  const { theme } = useTheme()
  
  // State to track which task series are hidden
  const [hiddenTasks, setHiddenTasks] = useState<Set<string>>(new Set())
  
  // Handle legend click to toggle series visibility
  const handleLegendClick = (data: any) => {
    const dataKey = data.dataKey || data.value
    if (!dataKey) return
    
    const newHiddenTasks = new Set(hiddenTasks)
    if (hiddenTasks.has(dataKey)) {
      newHiddenTasks.delete(dataKey)
    } else {
      newHiddenTasks.add(dataKey)
    }
    setHiddenTasks(newHiddenTasks)
  }
  
  // Prepare chart data with formatted dates
  const chartData = useMemo(() => {
    if (data.length === 0) return []
    
    return data.map((item, index) => {
      const date = new Date(item.date)
      let displayDate: string
      
      if (granularity === 'hour') {
        // For hourly data, check if we need to show day context
        const prevDate = index > 0 ? new Date(data[index - 1].date) : null
        const isNewDay = !prevDate || date.getDate() !== prevDate.getDate()
        
        if (isNewDay) {
          // Show day and hour for the first hour of each day
          displayDate = date.toLocaleDateString('en-US', { 
            month: 'short', 
            day: 'numeric'
          }) + ` ${date.getHours()}:00`
        } else {
          // Just show hour for subsequent hours
          displayDate = `${date.getHours()}:00`
        }
      } else {
        // For daily data, show month and day
        displayDate = date.toLocaleDateString('en-US', { 
          month: 'short', 
          day: 'numeric'
        })
      }
      
      return {
        ...item,
        displayDate
      }
    })
  }, [data, granularity])

  // Color palette for different tasks
  const taskColors = [
    '#3b82f6', // Blue
    '#10b981', // Emerald
    '#f59e0b', // Amber
    '#ef4444', // Red
    '#8b5cf6', // Violet
    '#06b6d4', // Cyan
    '#84cc16', // Lime
    '#f97316', // Orange
    '#ec4899', // Pink
    '#6366f1', // Indigo
  ]

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      const totalCost = payload.reduce((sum: number, item: any) => sum + (item.value || 0), 0)
      const data = payload[0].payload
      const date = new Date(data.date)
      
      // Format date with hour if granularity is hourly
      let dateString = date.toLocaleDateString('en-US', { 
        weekday: 'long', 
        year: 'numeric', 
        month: 'long', 
        day: 'numeric' 
      })
      
      if (granularity === 'hour') {
        const hour = date.getHours().toString().padStart(2, '0')
        dateString += ` at ${hour}:00`
      }
      
      return (
        <div className="bg-background border rounded-lg shadow-md p-3">
          <p className="font-medium">{dateString}</p>
          <div className="space-y-1 mt-2 text-sm">
            <p className="flex justify-between gap-4 font-medium border-b pb-1">
              <span>Total:</span>
              <span>{formatCurrencyAutoPrecision(totalCost, currency)}</span>
            </p>
            {payload
              .filter((item: any) => item.value > 0)
              .sort((a: any, b: any) => b.value - a.value)
              .map((item: any, index: number) => (
                <p key={index} className="flex justify-between gap-4">
                  <span className="flex items-center gap-2">
                    <div 
                      className="w-3 h-3 rounded-sm" 
                      style={{ backgroundColor: item.color }}
                    />
                    {item.dataKey}:
                  </span>
                  <span className="font-medium">{formatCurrencyAutoPrecision(item.value, currency)}</span>
                </p>
              ))}
          </div>
        </div>
      )
    }
    return null
  }

  return (
    <div className="h-80 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart
          data={chartData}
          margin={{
            top: 20,
            right: 30,
            left: 20,
            bottom: 5,
          }}
        >
          <CartesianGrid 
            strokeDasharray="3 3" 
            stroke={theme === 'dark' ? '#374151' : '#e5e7eb'}
          />
          <XAxis 
            dataKey="displayDate"
            stroke={theme === 'dark' ? '#9ca3af' : '#6b7280'}
            fontSize={12}
            tick={{ fill: theme === 'dark' ? '#9ca3af' : '#6b7280' }}
          />
          <YAxis
            stroke={theme === 'dark' ? '#9ca3af' : '#6b7280'}
            fontSize={12}
            tick={{ fill: theme === 'dark' ? '#9ca3af' : '#6b7280' }}
            tickFormatter={(value) => formatCurrencyAutoPrecision(value, currency)}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend 
            wrapperStyle={{
              fontSize: '12px',
              paddingTop: '10px',
              cursor: 'pointer'
            }}
            onClick={handleLegendClick}
            iconType="rect"
            formatter={(value, entry) => (
              <span 
                style={{ 
                  opacity: hiddenTasks.has(value) ? 0.4 : 1,
                  textDecoration: hiddenTasks.has(value) ? 'line-through' : 'none'
                }}
              >
                {value}
              </span>
            )}
          />
          {taskNames.map((taskName, index) => (
            <Area
              key={taskName}
              type="monotone"
              dataKey={taskName}
              stackId="1"
              stroke={taskColors[index % taskColors.length]}
              fill={taskColors[index % taskColors.length]}
              fillOpacity={0.6}
              strokeWidth={1}
              hide={hiddenTasks.has(taskName)}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}