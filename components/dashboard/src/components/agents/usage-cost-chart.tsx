'use client'

import { useMemo } from 'react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { useTheme } from 'next-themes'
import { formatCurrencyAutoPrecision } from '@/lib/currency'

interface UsageData {
  date: string
  cost: number
  tokens: number
  tasks: number
  avgResponseTime: number
  errors: number
}

interface UsageCostChartProps {
  data: UsageData[]
  granularity?: string
  currency?: string
}

export function UsageCostChart({ data, granularity = 'day', currency = 'USD' }: UsageCostChartProps) {
  const { theme } = useTheme()
  
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

  // Line chart color
  const lineColor = '#22c55e'

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
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
            <p className="flex justify-between gap-4">
              <span>Cost:</span>
              <span className="font-medium">{formatCurrencyAutoPrecision(data.cost, currency)}</span>
            </p>
            <p className="flex justify-between gap-4">
              <span>Tokens:</span>
              <span className="font-medium">{data.tokens.toLocaleString()}</span>
            </p>
            {data.errors > 0 && (
              <p className="flex justify-between gap-4 text-red-500">
                <span>Errors:</span>
                <span className="font-medium">{data.errors}</span>
              </p>
            )}
          </div>
        </div>
      )
    }
    return null
  }

  // Line chart styling
  const strokeWidth = 3

  return (
    <div className="h-80 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
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
            tickFormatter={(value) => `$${value}`}
          />
          <Tooltip content={<CustomTooltip />} />
          <Line 
            type="monotone" 
            dataKey="cost" 
            stroke={lineColor} 
            strokeWidth={strokeWidth}
            dot={{ fill: lineColor, strokeWidth: 0, r: 4 }}
            activeDot={{ r: 6, fill: lineColor }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}