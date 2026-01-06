import { notFound } from 'next/navigation'
import { AgentTraceViewer } from '@/components/agents/agent-trace-viewer'

interface PageProps {
  params: Promise<{
    name: string
    agentName: string 
    traceId: string
  }>
}

// Convert trace ID to execution ID format expected by the API
function traceIdToExecutionId(traceId: string): string {
  return `exec_${traceId.substring(0, 8)}`
}

export default async function TracePage({ params }: PageProps) {
  const { name: clusterName, agentName, traceId } = await params

  // Create a mock execution object for the trace viewer
  // In a real implementation, you might fetch this from an API
  const execution = {
    traceId: traceId,
    executionId: traceIdToExecutionId(traceId),
    startTime: new Date(), // These would be fetched from the API
    endTime: new Date(),
    duration: 0,
    status: 'success' as const,
    rootSpanName: '',
    spanCount: 0
  }

  return (
    <div className="container mx-auto p-6">
      <div className="mb-6">
        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-2">
          <a href={`/clusters/${clusterName}/agents`} className="hover:text-foreground">
            Agents
          </a>
          <span>/</span>
          <a href={`/clusters/${clusterName}/agents/${agentName}`} className="hover:text-foreground">
            {agentName}
          </a>
          <span>/</span>
          <a href={`/clusters/${clusterName}/agents/${agentName}/traces`} className="hover:text-foreground">
            Traces
          </a>
          <span>/</span>
          <span className="font-mono text-xs">{traceId}</span>
        </div>
        <h1 className="text-2xl font-bold">Trace Details</h1>
        <p className="text-muted-foreground">
          Detailed view for trace: <span className="font-mono text-sm">{traceId}</span>
        </p>
      </div>

      <AgentTraceViewer
        execution={execution}
        clusterName={clusterName}
        agentName={agentName}
      />
    </div>
  )
}