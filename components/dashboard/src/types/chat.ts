// Chat-related types for agent communication

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string
  agentName?: string
  status?: 'sending' | 'sent' | 'delivered' | 'error'
  // Optional fields for separating thinking content from response
  thinkingContent?: string[]
  responseContent?: string
  hasThinking?: boolean
}

export interface ChatConversation {
  id: string
  agentName: string
  clusterName: string
  messages: ChatMessage[]
  startedAt: string
  lastMessageAt: string
}

export interface AgentChatInfo {
  instructions?: string
  executionMode?: string
  modelRefs: Array<{ name: string; namespace?: string; role?: string }>
  toolRefs: Array<{ name: string; namespace?: string; enabled?: boolean }>
  personaRefs: Array<{ name: string; namespace?: string }>
  networking?: {
    port: number
    enabled: boolean
  }
}

export interface AgentChatStatus {
  agentName: string
  chatAvailable: boolean
  status: 'Ready' | 'Running' | 'Pending' | 'Failed' | 'Unknown'
  agentInfo: AgentChatInfo
}

export interface ChatRequest {
  message: string
  conversation?: ChatMessage[]
}

export interface ChatResponse {
  success: boolean
  message: ChatMessage
  agentName: string
  agentStatus: string
  agentInfo: AgentChatInfo
}

export interface ChatError {
  error: string
  message: string
  code?: string
}

// Hook return types
export interface UseChatReturn {
  messages: ChatMessage[]
  sendMessage: (content: string) => Promise<void>
  isLoading: boolean
  error: string | null
  clearMessages: () => void
  agentStatus: AgentChatStatus | null
  isConnected: boolean
}

export interface UseAgentChatStatusReturn {
  status: AgentChatStatus | null
  isLoading: boolean
  error: string | null
  refetch: () => void
}