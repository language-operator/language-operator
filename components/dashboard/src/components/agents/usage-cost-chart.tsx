'use client'

import { useMemo } from 'react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { useTheme } from 'next-themes'

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
}

export function UsageCostChart({ data }: UsageCostChartProps) {
  const { theme } = useTheme()
  
  // Prepare chart data with formatted dates
  const chartData = useMemo(() => {
    return data.map(item => ({
      ...item,
      displayDate: new Date(item.date).toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric'
      })
    }))
  }, [data])

  // Line chart color
  const lineColor = '#22c55e'

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload
      return (
        <div className="bg-background border rounded-lg shadow-md p-3">
          <p className="font-medium">{new Date(data.date).toLocaleDateString('en-US', { 
            weekday: 'long', 
            year: 'numeric', 
            month: 'long', 
            day: 'numeric' 
          })}</p>
          <div className="space-y-1 mt-2 text-sm">
            <p className="flex justify-between gap-4">
              <span>Cost:</span>
              <span className="font-medium">${data.cost.toFixed(2)}</span>
            </p>
            <p className="flex justify-between gap-4">
              <span>Tokens:</span>
              <span className="font-medium">{data.tokens.toLocaleString()}</span>
            </p>
            <p className="flex justify-between gap-4">
              <span>Tasks:</span>
              <span className="font-medium">{data.tasks}</span>
            </p>
            <p className="flex justify-between gap-4">
              <span>Avg Response:</span>
              <span className="font-medium">{data.avgResponseTime}s</span>
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