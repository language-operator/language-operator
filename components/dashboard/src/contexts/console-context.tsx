'use client'

import React, { createContext, useContext, useState, useCallback } from 'react'
import { ChatMessage } from '@/types/chat'
import { saveLastConversation, clearLastConversation } from '@/lib/conversation-storage'
import { fetchWithOrganization } from '@/lib/api-client'

interface ConversationState {
  agentName: string
  clusterName: string
  messages: ChatMessage[]
  isLoading: boolean
  error: string | null
  lastActivity: Date
}

interface ConsoleContextType {
  // Selected agent
  selectedAgent: string | null
  selectedCluster: string | null
  setSelectedAgent: (agentName: string | null, clusterName: string | null) => void
  startNewConversation: (agentName: string, clusterName: string) => void

  // Conversation state
  activeConversationId: string | null
  conversations: Map<string, ConversationState>
  getActiveConversation: () => ConversationState | null
  addMessage: (agentName: string, message: ChatMessage) => void
  updateMessage: (agentName: string, messageId: string, updates: Partial<ChatMessage>) => void
  setLoading: (agentName: string, isLoading: boolean) => void
  setError: (agentName: string, error: string | null) => void

  // Database conversation ID tracking
  conversationDbId: string | null
  setConversationDbId: (id: string | null) => void
  loadConversation: (conversationId: string, agentName: string, clusterName: string) => Promise<void>
  deleteConversation: (conversationId: string) => Promise<void>
  refreshConversationList: () => void
  conversationListRefreshTrigger: number

  // Workspace visibility
  isWorkspaceVisible: boolean
  toggleWorkspace: () => void
  setWorkspaceVisible: (visible: boolean) => void

  // Conversation sidebar visibility
  isConversationSidebarVisible: boolean
  toggleConversationSidebar: () => void
  setConversationSidebarVisible: (visible: boolean) => void
}

const ConsoleContext = createContext<ConsoleContextType | undefined>(undefined)

interface ConsoleProviderProps {
  children: React.ReactNode
  initialAgent?: string | null
  initialCluster?: string | null
}

export function ConsoleProvider({
  children,
  initialAgent = null,
  initialCluster = null
}: ConsoleProviderProps) {
  const [selectedAgent, setSelectedAgentState] = useState<string | null>(initialAgent)
  const [selectedCluster, setSelectedClusterState] = useState<string | null>(initialCluster)
  const [activeConversationId, setActiveConversationId] = useState<string | null>(
    initialAgent && initialCluster ? `${initialCluster}/${initialAgent}` : null
  )
  const [conversations, setConversations] = useState<Map<string, ConversationState>>(() => {
    const map = new Map()
    if (initialAgent && initialCluster) {
      const key = `${initialCluster}/${initialAgent}`
      map.set(key, {
        agentName: initialAgent,
        clusterName: initialCluster,
        messages: [],
        isLoading: false,
        error: null,
        lastActivity: new Date(),
      })
    }
    return map
  })
  const [conversationDbId, setConversationDbId] = useState<string | null>(null)
  const [conversationListRefreshTrigger, setConversationListRefreshTrigger] = useState(0)
  const [isWorkspaceVisible, setWorkspaceVisible] = useState(true)
  const [isConversationSidebarVisible, setConversationSidebarVisible] = useState(true)

  const setSelectedAgent = useCallback(
    (agentName: string | null, clusterName: string | null) => {
      setSelectedAgentState(agentName)
      setSelectedClusterState(clusterName)

      if (agentName && clusterName) {
        // Initialize conversation state if it doesn't exist
        const key = `${clusterName}/${agentName}`
        setConversations((prev) => {
          const newMap = new Map(prev)
          if (!newMap.has(key)) {
            newMap.set(key, {
              agentName,
              clusterName,
              messages: [],
              isLoading: false,
              error: null,
              lastActivity: new Date(),
            })
          }
          return newMap
        })
        setActiveConversationId(key)
      } else {
        setActiveConversationId(null)
      }
    },
    []
  )

  const startNewConversation = useCallback(
    (agentName: string, clusterName: string) => {
      // Clear the conversation database ID to indicate this is a new conversation
      setConversationDbId(null)
      
      // Set the selected agent and cluster
      setSelectedAgentState(agentName)
      setSelectedClusterState(clusterName)

      // Force create a fresh conversation state with empty messages
      const key = `${clusterName}/${agentName}`
      setConversations((prev) => {
        const newMap = new Map(prev)
        newMap.set(key, {
          agentName,
          clusterName,
          messages: [], // Always start with empty messages
          isLoading: false,
          error: null,
          lastActivity: new Date(),
        })
        return newMap
      })
      
      setActiveConversationId(key)
    },
    []
  )

  const getActiveConversation = useCallback((): ConversationState | null => {
    if (!activeConversationId) return null
    return conversations.get(activeConversationId) || null
  }, [activeConversationId, conversations])

  const addMessage = useCallback((agentName: string, message: ChatMessage) => {
    setConversations((prev) => {
      const newMap = new Map(prev)
      const key = Array.from(newMap.keys()).find((k) => k.endsWith(`/${agentName}`))
      if (key) {
        const conv = newMap.get(key)
        if (conv) {
          // Check if message with same ID already exists
          const existingIndex = conv.messages.findIndex(m => m.id === message.id)
          let updatedMessages
          
          if (existingIndex >= 0) {
            // Replace existing message
            updatedMessages = [...conv.messages]
            updatedMessages[existingIndex] = message
          } else {
            // Add new message
            updatedMessages = [...conv.messages, message]
          }
          
          newMap.set(key, {
            ...conv,
            messages: updatedMessages,
            lastActivity: new Date(),
          })
        }
      }
      return newMap
    })
  }, [])

  const updateMessage = useCallback((agentName: string, messageId: string, updates: Partial<ChatMessage>) => {
    setConversations((prev) => {
      const newMap = new Map(prev)
      const key = Array.from(newMap.keys()).find((k) => k.endsWith(`/${agentName}`))
      if (key) {
        const conv = newMap.get(key)
        if (conv) {
          const existingIndex = conv.messages.findIndex(m => m.id === messageId)
          if (existingIndex >= 0) {
            const updatedMessages = [...conv.messages]
            updatedMessages[existingIndex] = {
              ...updatedMessages[existingIndex],
              ...updates
            }
            
            newMap.set(key, {
              ...conv,
              messages: updatedMessages,
              lastActivity: new Date(),
            })
          }
        }
      }
      return newMap
    })
  }, [])

  const setLoading = useCallback((agentName: string, isLoading: boolean) => {
    setConversations((prev) => {
      const newMap = new Map(prev)
      const key = Array.from(newMap.keys()).find((k) => k.endsWith(`/${agentName}`))
      if (key) {
        const conv = newMap.get(key)
        if (conv) {
          newMap.set(key, {
            ...conv,
            isLoading,
          })
        }
      }
      return newMap
    })
  }, [])

  const setError = useCallback((agentName: string, error: string | null) => {
    setConversations((prev) => {
      const newMap = new Map(prev)
      const key = Array.from(newMap.keys()).find((k) => k.endsWith(`/${agentName}`))
      if (key) {
        const conv = newMap.get(key)
        if (conv) {
          newMap.set(key, {
            ...conv,
            error,
          })
        }
      }
      return newMap
    })
  }, [])

  const toggleWorkspace = useCallback(() => {
    setWorkspaceVisible((prev) => !prev)
  }, [])

  const toggleConversationSidebar = useCallback(() => {
    setConversationSidebarVisible((prev) => !prev)
  }, [])

  const refreshConversationList = useCallback(() => {
    setConversationListRefreshTrigger((prev) => prev + 1)
  }, [])

  const loadConversation = useCallback(
    async (conversationId: string, agentName: string, clusterName: string) => {
      try {
        // First, validate that the agent still exists
        const agentResponse = await fetchWithOrganization(`/api/clusters/${clusterName}/agents/${agentName}`)
        if (!agentResponse.ok) {
          throw new Error(`Agent "${agentName}" no longer exists in cluster "${clusterName}"`)
        }
        
        // Set the selected agent and cluster
        setSelectedAgentState(agentName)
        setSelectedClusterState(clusterName)
        setConversationDbId(conversationId)
        
        // Save to localStorage for auto-restore
        saveLastConversation(conversationId, agentName, clusterName)

        const key = `${clusterName}/${agentName}`
        setActiveConversationId(key)

        // Fetch messages from database
        const response = await fetch(`/api/conversations/${conversationId}/messages`)

        if (!response.ok) {
          throw new Error('Failed to load conversation messages')
        }

        const data = await response.json()

        // Convert database messages to ChatMessage format
        const messages = data.messages.map((msg: any) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          timestamp: new Date(msg.timestamp),
          ...(msg.toolCalls && { toolCalls: msg.toolCalls }),
          ...(msg.metadata && { metadata: msg.metadata }),
          // Include parsed thinking content fields if they exist
          ...(msg.thinkingContent && { thinkingContent: msg.thinkingContent }),
          ...(msg.responseContent && { responseContent: msg.responseContent }),
          ...(msg.hasThinking !== undefined && { hasThinking: msg.hasThinking }),
        }))

        // Update conversation state with loaded messages
        setConversations((prev) => {
          const newMap = new Map(prev)
          newMap.set(key, {
            agentName,
            clusterName,
            messages,
            isLoading: false,
            error: null,
            lastActivity: new Date(),
          })
          return newMap
        })
      } catch (error) {
        console.error('Error loading conversation:', error)
        const key = `${clusterName}/${agentName}`
        
        // Initialize with error state for orphaned conversations
        setConversations((prev) => {
          const newMap = new Map(prev)
          newMap.set(key, {
            agentName,
            clusterName,
            messages: [],
            isLoading: false,
            error: error instanceof Error ? error.message : 'Failed to load conversation',
            lastActivity: new Date(),
          })
          return newMap
        })
        
        // For orphaned conversations, don't re-throw to prevent unhandled errors
        // Callers can check the error state in the conversation object
        if (error instanceof Error && error.message.includes('no longer exists')) {
          // Clear from localStorage since this conversation is orphaned
          clearLastConversation()
          console.warn('Orphaned conversation detected and cleared from localStorage:', error.message)
        } else {
          // Re-throw other types of errors for proper handling
          throw error
        }
      }
    },
    []
  )

  const deleteConversation = useCallback(
    async (conversationId: string) => {
      try {
        const response = await fetch(`/api/conversations/${conversationId}`, {
          method: 'DELETE',
        })

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.error || 'Failed to delete conversation')
        }

        // If we're deleting the currently active conversation, clear the selection
        if (conversationDbId === conversationId) {
          setSelectedAgent(null, null)
          setConversationDbId(null)
          setActiveConversationId(null)
          
          // Clear from localStorage since this conversation no longer exists
          clearLastConversation()
        }

        // Clear the conversation from local state
        setConversations((prev) => {
          const newMap = new Map(prev)
          // Find and remove the conversation by ID (we need to match against conversationDbId)
          for (const [key, conv] of newMap.entries()) {
            // Note: This is a limitation - we don't have a direct mapping from conversation key to DB ID
            // But since we're refreshing the list anyway, this is okay
          }
          return newMap
        })

        // Refresh the conversation list to get updated data
        refreshConversationList()
      } catch (error) {
        console.error('Error deleting conversation:', error)
        throw error
      }
    },
    [conversationDbId, refreshConversationList]
  )

  const value: ConsoleContextType = {
    selectedAgent,
    selectedCluster,
    setSelectedAgent,
    startNewConversation,
    activeConversationId,
    conversations,
    getActiveConversation,
    addMessage,
    updateMessage,
    setLoading,
    setError,
    conversationDbId,
    setConversationDbId,
    loadConversation,
    deleteConversation,
    refreshConversationList,
    conversationListRefreshTrigger,
    isWorkspaceVisible,
    toggleWorkspace,
    setWorkspaceVisible,
    isConversationSidebarVisible,
    toggleConversationSidebar,
    setConversationSidebarVisible,
  }

  return (
    <ConsoleContext.Provider value={value}>{children}</ConsoleContext.Provider>
  )
}

export function useConsole() {
  const context = useContext(ConsoleContext)
  if (context === undefined) {
    throw new Error('useConsole must be used within a ConsoleProvider')
  }
  return context
}
