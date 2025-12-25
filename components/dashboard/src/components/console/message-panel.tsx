'use client'

import { useConsole } from '@/contexts/console-context'
import { MessageHeader } from './message-header'
import { MessageStream } from './message-stream'
import { WelcomeMessage } from './welcome-message'
import { ChatMessageInput } from '@/components/agents/chat-message-input'
import { useCallback } from 'react'
import { fetchWithOrganization } from '@/lib/api-client'

export function MessagePanel() {
  const {
    selectedAgent,
    selectedCluster,
    getActiveConversation,
    addMessage,
    updateMessage,
    setLoading,
    setError,
    conversationDbId,
    setConversationDbId,
    refreshConversationList,
  } = useConsole()

  const conversation = getActiveConversation()

  const handleSendMessage = useCallback(
    async (content: string) => {
      if (!selectedAgent || !selectedCluster || !conversation) return

      // Add user message optimistically
      const userMessage = {
        id: Date.now().toString(),
        role: 'user' as const,
        content,
        timestamp: new Date().toISOString(),
        status: 'sending' as const,
      }

      addMessage(selectedAgent, userMessage)
      setLoading(selectedAgent, true)
      setError(selectedAgent, null)

      try {
        // Create conversation in database if it doesn't exist
        let dbConversationId = conversationDbId

        if (!dbConversationId) {
          const createResponse = await fetchWithOrganization('/api/conversations', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              agentName: selectedAgent,
              clusterName: selectedCluster,
            }),
          })

          if (!createResponse.ok) {
            throw new Error('Failed to create conversation')
          }

          const createData = await createResponse.json()
          dbConversationId = createData.conversation.id
          setConversationDbId(dbConversationId)

          // Refresh conversation list to show the new conversation
          refreshConversationList()
        }

        // Save user message to database
        const saveUserResponse = await fetchWithOrganization(
          `/api/conversations/${dbConversationId}/messages`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              role: 'user',
              content,
            }),
          }
        )

        if (!saveUserResponse.ok) {
          throw new Error('Failed to save message')
        }

        // Send message to agent
        const chatResponse = await fetchWithOrganization(
          `/api/clusters/${selectedCluster}/agents/${selectedAgent}/chat`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              message: content,
              conversation: conversation.messages.filter(msg => msg.role !== 'system')
            }),
          }
        )

        if (!chatResponse.ok) {
          throw new Error('Failed to send message to agent')
        }

        const response = await chatResponse.json()

        // Update user message to delivered
        updateMessage(selectedAgent, userMessage.id, {
          status: 'delivered',
        })

        // Add assistant response and save to database
        if (response.message) {
          const assistantMessage = {
            id: (Date.now() + 1).toString(),
            role: 'assistant' as const,
            content: response.message.content,
            timestamp: new Date().toISOString(),
            agentName: response.agentName,
            // Include parsed thinking content fields
            thinkingContent: response.message.thinkingContent,
            responseContent: response.message.responseContent,
            hasThinking: response.message.hasThinking,
          }

          addMessage(selectedAgent, assistantMessage)

          // Save assistant message to database
          await fetchWithOrganization(
            `/api/conversations/${dbConversationId}/messages`,
            {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                role: 'assistant',
                content: response.message.content,
              }),
            }
          )
        }
      } catch (err) {
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to send message'
        setError(selectedAgent, errorMessage)

        // Mark user message as error
        updateMessage(selectedAgent, userMessage.id, {
          status: 'error',
        })
      } finally {
        setLoading(selectedAgent, false)
      }
    },
    [
      selectedAgent,
      selectedCluster,
      conversation,
      addMessage,
      updateMessage,
      setLoading,
      setError,
      conversationDbId,
      setConversationDbId,
      refreshConversationList,
    ]
  )

  if (!selectedAgent || !selectedCluster) {
    return <WelcomeMessage />
  }

  return (
    <div className="flex flex-col h-full">
      <MessageHeader />

      <div className="flex-1 overflow-hidden">
        <MessageStream />
      </div>

      <div className="border-t border-stone-800/80 dark:border-stone-600/80">
        <ChatMessageInput
          onSendMessage={handleSendMessage}
          placeholder={`Message ${selectedAgent}...`}
          disabled={conversation?.isLoading}
        />
      </div>
    </div>
  )
}
