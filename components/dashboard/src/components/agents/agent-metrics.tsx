'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Activity, TrendingUp, Zap, DollarSign } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'

interface AgentMetricsProps {
  agent: LanguageAgent
}

export function AgentMetrics({ agent }: AgentMetricsProps) {
  const metrics = agent.status?.metrics

  return (
    <div className="space-y-6">
      {/* Key Metrics Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Execution Count</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {agent.status?.executionCount?.toLocaleString() ?? 0}
            </div>
            <p className="text-xs text-muted-foreground">
              Total executions
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.successRate ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Success percentage
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.averageLatency ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Average response time
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {metrics?.costMetrics?.totalCost ?? 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              {metrics?.costMetrics?.currency ?? 'USD'} total
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Metrics */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Performance Metrics</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Error Rate</p>
              <p className="text-sm">{metrics?.errorRate ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Total Requests</p>
              <p className="text-sm">{metrics?.totalRequests?.toLocaleString() ?? 'N/A'}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Average Latency</p>
              <p className="text-sm">{metrics?.averageLatency ?? 'N/A'}</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Cost Metrics</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {metrics?.costMetrics ? (
              <>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Total Cost</p>
                  <p className="text-sm">
                    {metrics.costMetrics.totalCost} {metrics.costMetrics.currency}
                  </p>
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Cost per Execution</p>
                  <p className="text-sm">
                    {metrics.costMetrics.costPerExecution} {metrics.costMetrics.currency}
                  </p>
                </div>
                {metrics.costMetrics.billingPeriod && (
                  <div>
                    <p className="text-sm font-medium text-muted-foreground">Billing Period</p>
                    <p className="text-sm">{metrics.costMetrics.billingPeriod}</p>
                  </div>
                )}
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No cost metrics available</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
