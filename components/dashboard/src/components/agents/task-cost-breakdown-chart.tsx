'use client'

import { useMemo } from 'react'
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { useTheme } from 'next-themes'

interface TaskCostData {
  date: string
  [taskName: string]: string | number // Dynamic task names as keys with cost values
}

interface TaskCostBreakdownChartProps {
  data: TaskCostData[]
  taskNames: string[]
}

export function TaskCostBreakdownChart({ data, taskNames }: TaskCostBreakdownChartProps) {
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
      
      return (
        <div className="bg-background border rounded-lg shadow-md p-3">
          <p className="font-medium">{label}</p>
          <div className="space-y-1 mt-2 text-sm">
            <p className="flex justify-between gap-4 font-medium border-b pb-1">
              <span>Total:</span>
              <span>${totalCost.toFixed(2)}</span>
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
                  <span className="font-medium">${item.value.toFixed(2)}</span>
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
            tickFormatter={(value) => `$${value}`}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend 
            wrapperStyle={{
              fontSize: '12px',
              paddingTop: '10px'
            }}
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
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}