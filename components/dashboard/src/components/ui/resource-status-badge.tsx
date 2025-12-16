'use client'

import { AnimatedStatus, AnimatedStatusProps } from './animated-status'

export interface ResourceStatusBadgeProps extends Omit<AnimatedStatusProps, 'status'> {
  status?: string
  phase?: string
}

export function ResourceStatusBadge({ status, phase, ...props }: ResourceStatusBadgeProps) {
  // Prefer phase over status, fallback to status, then 'Unknown'
  const displayStatus = phase || status || 'Unknown'
  
  return <AnimatedStatus status={displayStatus} {...props} />
}

// Cluster-specific status badge
export interface ClusterStatusBadgeProps extends Omit<ResourceStatusBadgeProps, 'status' | 'phase'> {
  cluster: {
    status?: {
      phase?: string
    }
  }
}

export function ClusterStatusBadge({ cluster, ...props }: ClusterStatusBadgeProps) {
  return (
    <ResourceStatusBadge 
      phase={cluster.status?.phase} 
      {...props} 
    />
  )
}

// Model-specific status badge
export interface ModelStatusBadgeProps extends Omit<ResourceStatusBadgeProps, 'status' | 'phase'> {
  model: {
    status?: {
      phase?: string
    }
  }
}

export function ModelStatusBadge({ model, ...props }: ModelStatusBadgeProps) {
  return (
    <ResourceStatusBadge 
      phase={model.status?.phase} 
      {...props} 
    />
  )
}

// Agent-specific status badge
export interface AgentStatusBadgeProps extends Omit<ResourceStatusBadgeProps, 'status' | 'phase'> {
  agent: {
    status?: {
      phase?: string
    }
  }
}

export function AgentStatusBadge({ agent, ...props }: AgentStatusBadgeProps) {
  return (
    <ResourceStatusBadge 
      phase={agent.status?.phase} 
      {...props} 
    />
  )
}

// Tool-specific status badge
export interface ToolStatusBadgeProps extends Omit<ResourceStatusBadgeProps, 'status' | 'phase'> {
  tool: {
    status?: {
      phase?: string
    }
  }
}

export function ToolStatusBadge({ tool, ...props }: ToolStatusBadgeProps) {
  return (
    <ResourceStatusBadge 
      phase={tool.status?.phase} 
      {...props} 
    />
  )
}

// Persona-specific status badge
export interface PersonaStatusBadgeProps extends Omit<ResourceStatusBadgeProps, 'status' | 'phase'> {
  persona: {
    status?: {
      phase?: string
    }
  }
}

export function PersonaStatusBadge({ persona, ...props }: PersonaStatusBadgeProps) {
  return (
    <ResourceStatusBadge 
      phase={persona.status?.phase} 
      {...props} 
    />
  )
}