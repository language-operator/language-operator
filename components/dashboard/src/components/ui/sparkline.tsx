'use client'

import { AreaChart, Area, ResponsiveContainer } from 'recharts'
import { useTheme } from 'next-themes'

interface SparklineProps {
  data: number[]
  color?: string
  className?: string
  showDots?: boolean
}

export function Sparkline({ data, color, className, showDots = false }: SparklineProps) {
  const { theme } = useTheme()
  
  // Convert array to chart data format
  const chartData = data.map((value, index) => ({
    index,
    value
  }))

  // Default colors based on theme
  const defaultColor = color || (theme === 'dark' ? '#60a5fa' : '#3b82f6')
  const strokeOpacity = 1.0
  const fillOpacity = 0.6
  
  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData}>
          <defs>
            <linearGradient id={`gradient-${defaultColor.replace('#', '')}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={defaultColor} stopOpacity={fillOpacity} />
              <stop offset="100%" stopColor={defaultColor} stopOpacity={0.2} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="value"
            stroke={defaultColor}
            strokeWidth={1.5}
            strokeOpacity={strokeOpacity}
            fill={`url(#gradient-${defaultColor.replace('#', '')})`}
            dot={showDots ? { fill: defaultColor, r: 1.5, strokeWidth: 0 } : false}
            activeDot={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}