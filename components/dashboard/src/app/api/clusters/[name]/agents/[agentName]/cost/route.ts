import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth/next'
import { authOptions } from '@/lib/auth'
import { z } from 'zod'
import { createClickHouseClient } from '@/lib/clickhouse-config'
import { k8sClient } from '@/lib/k8s-client'

// Validation schema for query parameters
const UsageQuerySchema = z.object({
  from: z.string().datetime().optional(),
  to: z.string().datetime().optional(),
  granularity: z.enum(['hour', 'day', 'week']).nullable().optional().default('day'),
})

// Types for our usage data
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
  [taskName: string]: string | number // Dynamic task names as keys
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

interface UsageResponse {
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

// ClickHouse client
const clickhouse = createClickHouseClient()

// Cache for model pricing data
const modelPricingCache = new Map<string, { inputTokenCost: number; outputTokenCost: number; currency: string; timestamp: number }>()

// Helper function to normalize model names for Kubernetes resource lookup
function normalizeModelName(telemetryModelName: string): string {
  // Convert colons to dashes for Kubernetes resource names
  // e.g., "qwen3-coder:30b" -> "qwen3-coder-30b"
  return telemetryModelName.replace(/:/g, '-')
}

// Helper function to get model pricing
async function getModelPricing(telemetryModelName: string, namespace: string): Promise<{ inputTokenCost: number; outputTokenCost: number; currency: string }> {
  const k8sModelName = normalizeModelName(telemetryModelName)
  const cacheKey = `${namespace}:${k8sModelName}`
  const cached = modelPricingCache.get(cacheKey)
  
  // Cache for 5 minutes
  if (cached && Date.now() - cached.timestamp < 5 * 60 * 1000) {
    return { inputTokenCost: cached.inputTokenCost, outputTokenCost: cached.outputTokenCost, currency: cached.currency }
  }
  
  const model = await k8sClient.getLanguageModel(namespace, k8sModelName)
  
  if (!model) {
    throw new Error(`LanguageModel '${k8sModelName}' (telemetry: '${telemetryModelName}') not found in namespace '${namespace}'`)
  }
  
  if (!model.spec.costTracking?.enabled) {
    throw new Error(`Cost tracking not enabled for model '${k8sModelName}'`)
  }
  
  const pricing = {
    inputTokenCost: model.spec.costTracking.inputTokenCost,
    outputTokenCost: model.spec.costTracking.outputTokenCost,
    currency: model.spec.costTracking.currency || 'USD'
  }
  
  // Cache the result
  modelPricingCache.set(cacheKey, {
    ...pricing,
    timestamp: Date.now()
  })
  
  return pricing
}

// Helper function to generate complete time intervals for padding
function generateTimeIntervals(from: Date, to: Date, granularity: string): Date[] {
  const intervals: Date[] = []
  let current = new Date(from)
  
  if (granularity === 'hour') {
    // Round to start of hour
    current.setMinutes(0, 0, 0)
    while (current <= to) {
      intervals.push(new Date(current)) // Create new Date object each time
      current = new Date(current.getTime() + 60 * 60 * 1000) // Add 1 hour
    }
  } else {
    // Round to start of day
    current.setHours(0, 0, 0, 0)
    while (current <= to) {
      intervals.push(new Date(current)) // Create new Date object each time
      current = new Date(current.getTime() + 24 * 60 * 60 * 1000) // Add 1 day
    }
  }
  
  return intervals
}

// Helper function to pad usage time series data
function padUsageTimeSeries(data: UsageDataPoint[], from: Date, to: Date, granularity: string): UsageDataPoint[] {
  const intervals = generateTimeIntervals(from, to, granularity)
  const dataMap = new Map<string, UsageDataPoint>()
  
  // Index existing data by timestamp
  data.forEach(point => {
    dataMap.set(point.timestamp, point)
  })
  
  // Generate complete series with zero padding
  return intervals.map(interval => {
    const timestamp = interval.toISOString()
    return dataMap.get(timestamp) || {
      timestamp,
      tasks: 0,
      errors: 0,
      avgResponseTime: 0,
      cost: 0,
      tokens: 0,
    }
  })
}

// Helper function to pad task breakdown data
function padTaskBreakdown(data: TaskCostDataPoint[], taskNames: string[], from: Date, to: Date, granularity: string) {
  const intervals = generateTimeIntervals(from, to, granularity)
  const dataMap = new Map<string, TaskCostDataPoint>()
  
  // Index existing data by timestamp
  data.forEach(point => {
    dataMap.set(point.timestamp, point)
  })
  
  // Generate complete series with zero padding
  const paddedData = intervals.map(interval => {
    const timestamp = interval.toISOString()
    const existingPoint = dataMap.get(timestamp)
    
    if (existingPoint) {
      return existingPoint
    } else {
      // Create zero-padded point with all task names
      const point: TaskCostDataPoint = { timestamp }
      taskNames.forEach(taskName => {
        point[taskName] = 0
      })
      return point
    }
  })
  
  return { data: paddedData, taskNames }
}

// Real ClickHouse query functions
async function queryUsageTimeSeries(agentName: string, from: Date, to: Date, granularity: string, namespace: string): Promise<{ data: UsageDataPoint[]; currency: string }> {
  const intervalFunc = granularity === 'hour' ? 'toStartOfHour' : 'toStartOfDay'
  
  const sql = `
    SELECT 
      ${intervalFunc}(Timestamp) as timeInterval,
      count() as tasks,
      countIf(StatusCode = 'STATUS_CODE_ERROR') as errors,
      avg(Duration) / 1000000 as avgResponseTime,
      sum(toUInt64OrZero(SpanAttributes['gen_ai.usage.input_tokens'])) as inputTokens,
      sum(toUInt64OrZero(SpanAttributes['gen_ai.usage.output_tokens'])) as outputTokens,
      SpanAttributes['gen_ai.request.model'] as modelName
    FROM langop.otel_traces
    WHERE Timestamp >= {from:DateTime64(9)}
      AND Timestamp <= {to:DateTime64(9)}
      AND SpanAttributes['agent.name'] = {agentName:String}
      AND SpanName != 'agent.reconcile'
      AND (length(SpanAttributes['gen_ai.usage.input_tokens']) > 0 OR length(SpanAttributes['gen_ai.usage.output_tokens']) > 0)
    GROUP BY timeInterval, modelName
    ORDER BY timeInterval ASC, modelName ASC
  `

  try {
    // Handle kubectl-proxy vs direct ClickHouse access
    if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy') && process.env.CLICKHOUSE_DIRECT_ACCESS !== 'true') {
      const baseUrl = `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`
      
      const sqlWithParams = sql
        .replace('{from:DateTime64(9)}', `fromUnixTimestamp64Nano(${from.getTime() * 1000000})`)
        .replace('{to:DateTime64(9)}', `fromUnixTimestamp64Nano(${to.getTime() * 1000000})`)
        .replace('{agentName:String}', `'${agentName}'`)

      console.log('Generated SQL query:', sqlWithParams)

      const response = await fetch(`${baseUrl}?database=${process.env.CLICKHOUSE_DATABASE || 'langop'}&query=${encodeURIComponent(sqlWithParams)}`, {
        headers: {
          'Authorization': `Basic ${Buffer.from(`${process.env.CLICKHOUSE_USERNAME || 'langop'}:${process.env.CLICKHOUSE_PASSWORD || 'langop'}`).toString('base64')}`
        }
      })
      
      if (!response.ok) {
        console.error('ClickHouse query failed:', response.status, response.statusText)
        return { data: [], currency: 'USD' }
      }
      
      const text = await response.text()
      const lines = text.trim().split('\n')
      
      if (lines.length === 0 || (lines.length === 1 && lines[0] === '')) {
        return { data: [], currency: 'USD' }
      }
      
      // Group by timeInterval and calculate costs with real pricing
      const dataByInterval = new Map<string, { tasks: number; errors: number; avgResponseTime: number; cost: number; tokens: number }>()
      let currency = 'USD' // Default fallback
      
      for (const line of lines) {
        const fields = line.split('\t')
        const timeInterval = fields[0]
        const tasks = parseInt(fields[1]) || 0
        const errors = parseInt(fields[2]) || 0 
        const avgResponseTime = parseFloat(fields[3]) || 0
        const inputTokens = parseInt(fields[4]) || 0
        const outputTokens = parseInt(fields[5]) || 0
        const modelName = fields[6]
        
        if (!modelName) continue
        
        try {
          const pricing = await getModelPricing(modelName, namespace)
          const cost = (inputTokens * pricing.inputTokenCost / 1000) + (outputTokens * pricing.outputTokenCost / 1000)
          
          const existing = dataByInterval.get(timeInterval) || { tasks: 0, errors: 0, avgResponseTime: 0, cost: 0, tokens: 0 }
          dataByInterval.set(timeInterval, {
            tasks: existing.tasks + tasks,
            errors: existing.errors + errors,
            avgResponseTime: ((existing.avgResponseTime * existing.tasks) + (avgResponseTime * tasks)) / (existing.tasks + tasks),
            cost: existing.cost + cost,
            tokens: existing.tokens + inputTokens + outputTokens
          })
        } catch (error) {
          console.warn(`Skipping cost calculation for model ${modelName}:`, error instanceof Error ? error.message : String(error))
          // Still include the data but with 0 cost
          const existing = dataByInterval.get(timeInterval) || { tasks: 0, errors: 0, avgResponseTime: 0, cost: 0, tokens: 0 }
          dataByInterval.set(timeInterval, {
            tasks: existing.tasks + tasks,
            errors: existing.errors + errors,
            avgResponseTime: ((existing.avgResponseTime * existing.tasks) + (avgResponseTime * tasks)) / (existing.tasks + tasks),
            cost: existing.cost,
            tokens: existing.tokens + inputTokens + outputTokens
          })
        }
      }
      
      return {
        data: Array.from(dataByInterval.entries()).map(([timeInterval, data]) => ({
          timestamp: new Date(timeInterval).toISOString(),
          ...data
        })),
        currency
      }
    } else {
      const resultSet = await clickhouse.query({
        query: sql,
        query_params: {
          from: from.getTime() * 1000000,
          to: to.getTime() * 1000000,
          agentName,
        },
      })
      const result = await resultSet.json()
      const data = result.data || []
      
      // Group by timeInterval and calculate costs with real pricing
      const dataByInterval = new Map<string, { tasks: number; errors: number; avgResponseTime: number; cost: number; tokens: number }>()
      let currency = 'USD' // Default fallback
      
      for (const row of data) {
        const rowData = row as any // Cast to any to access properties from ClickHouse result
        const timeInterval = rowData.timeInterval
        const tasks = parseInt(rowData.tasks) || 0
        const errors = parseInt(rowData.errors) || 0
        const avgResponseTime = parseFloat(rowData.avgResponseTime) || 0
        const inputTokens = parseInt(rowData.inputTokens) || 0
        const outputTokens = parseInt(rowData.outputTokens) || 0
        const modelName = rowData.modelName
        
        if (!modelName) continue
        
        try {
          const pricing = await getModelPricing(modelName, namespace)
          const cost = (inputTokens * pricing.inputTokenCost / 1000) + (outputTokens * pricing.outputTokenCost / 1000)
          currency = pricing.currency // Use the currency from the model
          
          const existing = dataByInterval.get(timeInterval) || { tasks: 0, errors: 0, avgResponseTime: 0, cost: 0, tokens: 0 }
          dataByInterval.set(timeInterval, {
            tasks: existing.tasks + tasks,
            errors: existing.errors + errors,
            avgResponseTime: ((existing.avgResponseTime * existing.tasks) + (avgResponseTime * tasks)) / (existing.tasks + tasks),
            cost: existing.cost + cost,
            tokens: existing.tokens + inputTokens + outputTokens
          })
        } catch (error) {
          console.warn(`Skipping cost calculation for model ${modelName}:`, error instanceof Error ? error.message : String(error))
          // Still include the data but with 0 cost
          const existing = dataByInterval.get(timeInterval) || { tasks: 0, errors: 0, avgResponseTime: 0, cost: 0, tokens: 0 }
          dataByInterval.set(timeInterval, {
            tasks: existing.tasks + tasks,
            errors: existing.errors + errors,
            avgResponseTime: ((existing.avgResponseTime * existing.tasks) + (avgResponseTime * tasks)) / (existing.tasks + tasks),
            cost: existing.cost,
            tokens: existing.tokens + inputTokens + outputTokens
          })
        }
      }
      
      return {
        data: Array.from(dataByInterval.entries()).map(([timeInterval, data]) => ({
          timestamp: new Date(timeInterval).toISOString(),
          ...data
        })),
        currency
      }
    }
  } catch (error) {
    console.error('Error querying ClickHouse usage data:', error)
    return { data: [], currency: 'USD' }
  }
}

async function queryTaskCostBreakdown(agentName: string, from: Date, to: Date, granularity: string, namespace: string) {
  const intervalFunc = granularity === 'hour' ? 'toStartOfHour' : 'toStartOfDay'
  
  const sql = `
    SELECT 
      ${intervalFunc}(Timestamp) as timeInterval,
      SpanAttributes['task.name'] as taskName,
      count() as taskCount,
      sum(toUInt64OrZero(SpanAttributes['gen_ai.usage.input_tokens'])) as inputTokens,
      sum(toUInt64OrZero(SpanAttributes['gen_ai.usage.output_tokens'])) as outputTokens,
      SpanAttributes['gen_ai.request.model'] as modelName
    FROM langop.otel_traces
    WHERE Timestamp >= {from:DateTime64(9)}
      AND Timestamp <= {to:DateTime64(9)}
      AND SpanAttributes['agent.name'] = {agentName:String}
      AND SpanName != 'agent.reconcile'
      AND (length(SpanAttributes['gen_ai.usage.input_tokens']) > 0 OR length(SpanAttributes['gen_ai.usage.output_tokens']) > 0)
      AND length(SpanAttributes['task.name']) > 0
    GROUP BY timeInterval, taskName, modelName
    ORDER BY timeInterval ASC, taskName ASC, modelName ASC
  `

  try {
    let rawData: any[] = []
    
    if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy') && process.env.CLICKHOUSE_DIRECT_ACCESS !== 'true') {
      const baseUrl = `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`
      
      const sqlWithParams = sql
        .replace('{from:DateTime64(9)}', `fromUnixTimestamp64Nano(${from.getTime() * 1000000})`)
        .replace('{to:DateTime64(9)}', `fromUnixTimestamp64Nano(${to.getTime() * 1000000})`)
        .replace('{agentName:String}', `'${agentName}'`)

      const response = await fetch(`${baseUrl}?database=${process.env.CLICKHOUSE_DATABASE || 'langop'}&query=${encodeURIComponent(sqlWithParams)}`, {
        headers: {
          'Authorization': `Basic ${Buffer.from(`${process.env.CLICKHOUSE_USERNAME || 'langop'}:${process.env.CLICKHOUSE_PASSWORD || 'langop'}`).toString('base64')}`
        }
      })
      
      if (!response.ok) {
        console.error('ClickHouse task breakdown query failed:', response.status, response.statusText)
        return { data: [], taskNames: [] }
      }
      
      const text = await response.text()
      const lines = text.trim().split('\n')
      
      if (lines.length === 0 || (lines.length === 1 && lines[0] === '')) {
        return { data: [], taskNames: [] }
      }
      
      rawData = []
      for (const line of lines) {
        const fields = line.split('\t')
        const timeInterval = fields[0]
        const taskName = fields[1]
        const taskCount = parseInt(fields[2]) || 0
        const inputTokens = parseInt(fields[3]) || 0
        const outputTokens = parseInt(fields[4]) || 0
        const modelName = fields[5]
        
        if (!modelName || !taskName) continue
        
        try {
          const pricing = await getModelPricing(modelName, namespace)
          const cost = (inputTokens * pricing.inputTokenCost / 1000) + (outputTokens * pricing.outputTokenCost / 1000)
          
          rawData.push({
            timeInterval,
            taskName,
            taskCount,
            cost,
          })
        } catch (error) {
          console.warn(`Skipping task cost calculation for model ${modelName}:`, error instanceof Error ? error.message : String(error))
          rawData.push({
            timeInterval,
            taskName,
            taskCount,
            cost: 0,
          })
        }
      }
    } else {
      const resultSet = await clickhouse.query({
        query: sql,
        query_params: {
          from: from.getTime() * 1000000,
          to: to.getTime() * 1000000,
          agentName,
        },
      })
      const result = await resultSet.json()
      const data = result.data || []
      
      rawData = []
      for (const row of data) {
        const rowData = row as any // Cast to any to access properties from ClickHouse result
        const timeInterval = rowData.timeInterval
        const taskName = rowData.taskName
        const taskCount = parseInt(rowData.taskCount) || 0
        const inputTokens = parseInt(rowData.inputTokens) || 0
        const outputTokens = parseInt(rowData.outputTokens) || 0
        const modelName = rowData.modelName
        
        if (!modelName || !taskName) continue
        
        try {
          const pricing = await getModelPricing(modelName, namespace)
          const cost = (inputTokens * pricing.inputTokenCost / 1000) + (outputTokens * pricing.outputTokenCost / 1000)
          
          rawData.push({
            timeInterval,
            taskName,
            taskCount,
            cost,
          })
        } catch (error) {
          console.warn(`Skipping task cost calculation for model ${modelName}:`, error instanceof Error ? error.message : String(error))
          rawData.push({
            timeInterval,
            taskName,
            taskCount,
            cost: 0,
          })
        }
      }
    }

    // Get unique task names
    const taskNames = [...new Set(rawData.map(row => row.taskName))].filter(name => name).sort()
    
    // Group by time interval and pivot by task name
    const timeIntervals = [...new Set(rawData.map(row => row.timeInterval))]
    const data: TaskCostDataPoint[] = timeIntervals.map(interval => {
      const point: TaskCostDataPoint = {
        timestamp: new Date(interval).toISOString()
      }
      
      // Add cost for each task
      taskNames.forEach(taskName => {
        const taskData = rawData.find(row => 
          (row.timeInterval === interval) && (row.taskName === taskName)
        )
        point[taskName] = taskData ? (taskData.cost || 0) : 0
      })
      
      return point
    })

    return { data, taskNames }
  } catch (error) {
    console.error('Error querying ClickHouse task breakdown:', error)
    return { data: [], taskNames: [] }
  }
}

function calculateMetrics(timeSeries: UsageDataPoint[]): UsageMetrics {
  const totals = timeSeries.reduce(
    (acc, point) => ({
      cost: acc.cost + point.cost,
      tokens: acc.tokens + point.tokens,
      tasks: acc.tasks + point.tasks,
      errors: acc.errors + point.errors,
      responseTime: acc.responseTime + point.avgResponseTime,
    }),
    { cost: 0, tokens: 0, tasks: 0, errors: 0, responseTime: 0 }
  )
  
  return {
    totalCost: parseFloat(totals.cost.toFixed(2)),
    totalTokens: totals.tokens,
    totalTasks: totals.tasks,
    totalErrors: totals.errors,
    avgResponseTime: parseFloat((totals.responseTime / timeSeries.length).toFixed(1)),
    costPerTask: totals.tasks > 0 ? parseFloat((totals.cost / totals.tasks).toFixed(3)) : 0,
    errorRate: totals.tasks > 0 ? parseFloat(((totals.errors / totals.tasks) * 100).toFixed(1)) : 0,
  }
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Await params in Next.js 15
    const { name, agentName } = await params
    
    // Verify authentication
    const session = await getServerSession(authOptions)
    if (!session) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    // Extract and validate query parameters
    const { searchParams } = new URL(request.url)
    const rawParams = {
      from: searchParams.get('from'),
      to: searchParams.get('to'),
      granularity: searchParams.get('granularity'),
    }
    
    console.log('Raw query parameters:', rawParams)
    
    const queryResult = UsageQuerySchema.safeParse(rawParams)

    if (!queryResult.success) {
      console.error('Query validation failed:', queryResult.error.issues)
      return NextResponse.json(
        { error: 'Invalid query parameters', details: queryResult.error.issues },
        { status: 400 }
      )
    }

    const { from: fromParam, to: toParam, granularity } = queryResult.data
    
    // Default to last 2 days if not provided
    const to = toParam ? new Date(toParam) : new Date()
    const from = fromParam ? new Date(fromParam) : new Date(Date.now() - 2 * 24 * 60 * 60 * 1000)

    // Validate date range
    if (from >= to) {
      return NextResponse.json(
        { error: 'Invalid date range: from must be before to' },
        { status: 400 }
      )
    }

    // Limit range to prevent excessive data
    const maxRangeMs = 90 * 24 * 60 * 60 * 1000 // 90 days
    if (to.getTime() - from.getTime() > maxRangeMs) {
      return NextResponse.json(
        { error: 'Date range too large. Maximum 90 days allowed.' },
        { status: 400 }
      )
    }

    // Automatically determine granularity based on date range if not specified
    const rangeDays = (to.getTime() - from.getTime()) / (24 * 60 * 60 * 1000)
    const autoGranularity = granularity || (rangeDays <= 7 ? 'hour' : 'day')

    // We need to determine the namespace for the agent to fetch model pricing
    // For now, we'll try to find the agent in multiple namespaces
    // TODO: Make this more robust by tracking agent->namespace mapping
    const possibleNamespaces = ['org-hdzkdwbv', 'default', 'language-operator']
    let agentNamespace = 'org-hdzkdwbv' // Default fallback
    
    // Query real ClickHouse data
    const [usageResult, rawTaskBreakdown] = await Promise.all([
      queryUsageTimeSeries(agentName, from, to, autoGranularity, agentNamespace),
      queryTaskCostBreakdown(agentName, from, to, autoGranularity, agentNamespace)
    ])
    
    // Pad both datasets to ensure consistent X-axis ranges
    const timeSeries = padUsageTimeSeries(usageResult.data, from, to, autoGranularity)
    const taskBreakdown = padTaskBreakdown(rawTaskBreakdown.data, rawTaskBreakdown.taskNames, from, to, autoGranularity)
    
    const metrics = calculateMetrics(timeSeries)

    const response: UsageResponse = {
      metrics,
      timeSeries,
      taskBreakdown,
      period: {
        from: from.toISOString(),
        to: to.toISOString(),
        granularity: autoGranularity,
      },
      currency: usageResult.currency,
    }

    return NextResponse.json(response)
  } catch (error) {
    console.error('Error fetching usage data:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}