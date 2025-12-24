'use client'

import { useState, useCallback, useRef, useEffect } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { 
  ChatMessage, 
  ChatConversation, 
  AgentChatStatus, 
  ChatResponse, 
  UseChatReturn,
  UseAgentChatStatusReturn 
} from '@/types/chat'

// Generate unique message ID
const generateMessageId = () => `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

// Hook to get agent chat status and capabilities
export function useAgentChatStatus(
  agentName: string, 
  clusterName: string
): UseAgentChatStatusReturn {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['agent-chat-status', clusterName, agentName],
    queryFn: async () => {
      const response = await fetchWithOrganization(
        `/api/clusters/${clusterName}/agents/${agentName}/chat`
      )
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to get agent chat status')
      }
      return response.json() as Promise<AgentChatStatus>
    },
    refetchInterval: 10000, // Refresh every 10 seconds
    staleTime: 5000, // Consider stale after 5 seconds
  })

  return {
    status: data || null,
    isLoading,
    error: error?.message || null,
    refetch
  }
}

// Main hook for agent chat functionality
export function useAgentChat(
  agentName: string, 
  clusterName: string,
  initialMessages: ChatMessage[] = []
): UseChatReturn {
  const [messages, setMessages] = useState<ChatMessage[]>(initialMessages)
  const [error, setError] = useState<string | null>(null)
  const conversationRef = useRef<ChatMessage[]>(messages)

  // Keep conversation ref in sync with messages
  useEffect(() => {
    conversationRef.current = messages
  }, [messages])

  // Get agent status
  const { status: agentStatus, refetch: refetchStatus } = useAgentChatStatus(agentName, clusterName)

  // Send message mutation
  const sendMessageMutation = useMutation({
    mutationFn: async (messageContent: string) => {
      const userMessage: ChatMessage = {
        id: generateMessageId(),
        role: 'user',
        content: messageContent,
        timestamp: new Date().toISOString(),
        status: 'sending'
      }

      // Add user message immediately for optimistic UI
      setMessages(prev => [...prev, userMessage])
      setError(null)

      try {
        const response = await fetchWithOrganization(
          `/api/clusters/${clusterName}/agents/${agentName}/chat`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              message: messageContent,
              conversation: conversationRef.current.filter(msg => msg.role !== 'system')
            }),
          }
        )

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.message || 'Failed to send message')
        }

        const data: ChatResponse = await response.json()

        // Update user message status
        setMessages(prev => 
          prev.map(msg => 
            msg.id === userMessage.id 
              ? { ...msg, status: 'delivered' as const }
              : msg
          )
        )

        // Add agent response
        const agentMessage: ChatMessage = {
          id: generateMessageId(),
          role: 'assistant',
          content: data.message.content,
          timestamp: data.message.timestamp,
          agentName: data.agentName,
          status: 'delivered'
        }

        setMessages(prev => [...prev, agentMessage])

        // Refresh agent status if needed
        if (data.agentStatus !== agentStatus?.status) {
          refetchStatus()
        }

        return data

      } catch (error) {
        console.error('Error sending message:', error)
        
        // Update user message status to error
        setMessages(prev => 
          prev.map(msg => 
            msg.id === userMessage.id 
              ? { ...msg, status: 'error' as const }
              : msg
          )
        )

        const errorMessage = error instanceof Error ? error.message : 'Failed to send message'
        setError(errorMessage)
        throw error
      }
    },
  })

  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim()) return
    if (!agentStatus?.chatAvailable) {
      setError('Agent is not available for chat')
      return
    }

    await sendMessageMutation.mutateAsync(content)
  }, [sendMessageMutation, agentStatus])

  const clearMessages = useCallback(() => {
    setMessages([])
    setError(null)
  }, [])

  return {
    messages,
    sendMessage,
    isLoading: sendMessageMutation.isPending,
    error,
    clearMessages,
    agentStatus,
    isConnected: agentStatus?.chatAvailable || false
  }
}