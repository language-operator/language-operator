'use client'

import { useEffect, useRef, useState } from 'react'
import { 
  Dialog, 
  DialogContent, 
  DialogHeader, 
  DialogTitle 
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  Bot, 
  MessageCircle, 
  X, 
  Trash2, 
  Cpu, 
  Wrench, 
  Users, 
  Zap,
  Activity,
  Clock
} from 'lucide-react'
import { useAgentChat } from '@/hooks/use-agent-chat'
import { AgentConnectionStatus } from './agent-connection-status'
import { ChatBubble, TypingIndicator } from './agent-chat-bubble'
import { ChatMessageInput } from './chat-message-input'
import { cn } from '@/lib/utils'

interface AgentChatDialogProps {
  isOpen: boolean
  onClose: () => void
  agentName: string
  clusterName: string
  agentDisplayName?: string
}

function getExecutionModeIcon(mode?: string) {
  switch (mode) {
    case 'autonomous':
      return <Zap className="h-3 w-3" />
    case 'interactive':
      return <Activity className="h-3 w-3" />
    case 'scheduled':
      return <Clock className="h-3 w-3" />
    default:
      return <Bot className="h-3 w-3" />
  }
}

export function AgentChatDialog({
  isOpen,
  onClose,
  agentName,
  clusterName,
  agentDisplayName
}: AgentChatDialogProps) {
  const {
    messages,
    sendMessage,
    isLoading,
    error,
    clearMessages,
    agentStatus,
    isConnected
  } = useAgentChat(agentName, clusterName)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [showTyping, setShowTyping] = useState(false)

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // Show typing indicator when sending message
  useEffect(() => {
    if (isLoading) {
      setShowTyping(true)
    } else {
      // Delay hiding typing to make it feel more natural
      const timer = setTimeout(() => setShowTyping(false), 500)
      return () => clearTimeout(timer)
    }
  }, [isLoading])

  // Auto-introduce agent when first opening chat
  useEffect(() => {
    if (isOpen && messages.length === 0 && agentStatus?.agentInfo?.instructions) {
      // Could add an automatic introduction message here
    }
  }, [isOpen, messages.length, agentStatus])

  const handleSendMessage = async (content: string) => {
    try {
      await sendMessage(content)
    } catch (error) {
      // Error is handled by the hook
      console.error('Failed to send message:', error)
    }
  }

  const handleClearChat = () => {
    if (confirm('Are you sure you want to clear this conversation?')) {
      clearMessages()
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent 
        className="max-w-4xl h-[80vh] flex flex-col p-0"
        showCloseButton={false}
      >
        {/* Header */}
        <DialogHeader className="px-6 py-4 border-b border-stone-200 dark:border-stone-700 shrink-0">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400">
                <Bot className="h-5 w-5" />
              </div>
              <div className="flex flex-col">
                <DialogTitle className="flex items-center gap-2 text-base">
                  Chat with {agentDisplayName || agentName}
                  <MessageCircle className="h-4 w-4" />
                </DialogTitle>
                <AgentConnectionStatus 
                  status={agentStatus} 
                  className="mt-1" 
                />
              </div>
            </div>
            
            <div className="flex items-center gap-2">
              {messages.length > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClearChat}
                  className="text-stone-600 hover:text-red-600"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={onClose}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Agent Info */}
          {agentStatus?.agentInfo && (
            <div className="flex flex-wrap gap-2 mt-3">
              {/* Execution Mode */}
              <Badge variant="outline" className="text-xs flex items-center gap-1">
                {getExecutionModeIcon(agentStatus.agentInfo.executionMode)}
                {agentStatus.agentInfo.executionMode || 'autonomous'}
              </Badge>

              {/* Models */}
              {agentStatus.agentInfo.modelRefs.slice(0, 2).map((model, index) => (
                <Badge key={index} variant="outline" className="text-xs flex items-center gap-1">
                  <Cpu className="h-3 w-3" />
                  {model.name}
                </Badge>
              ))}
              {agentStatus.agentInfo.modelRefs.length > 2 && (
                <Badge variant="outline" className="text-xs">
                  +{agentStatus.agentInfo.modelRefs.length - 2} models
                </Badge>
              )}

              {/* Tools */}
              {agentStatus.agentInfo.toolRefs.length > 0 && (
                <Badge variant="outline" className="text-xs flex items-center gap-1">
                  <Wrench className="h-3 w-3" />
                  {agentStatus.agentInfo.toolRefs.length} tools
                </Badge>
              )}

              {/* Personas */}
              {agentStatus.agentInfo.personaRefs.length > 0 && (
                <Badge variant="outline" className="text-xs flex items-center gap-1">
                  <Users className="h-3 w-3" />
                  {agentStatus.agentInfo.personaRefs.length} personas
                </Badge>
              )}
            </div>
          )}
        </DialogHeader>

        {/* Chat Area */}
        <div className="flex-1 flex flex-col min-h-0">
          <ScrollArea className="flex-1 p-6">
            {/* Welcome Message */}
            {messages.length === 0 && agentStatus?.agentInfo?.instructions && (
              <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg dark:bg-blue-900/20 dark:border-blue-800">
                <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
                  About this agent:
                </h4>
                <p className="text-sm text-blue-800 dark:text-blue-200 leading-relaxed">
                  {agentStatus.agentInfo.instructions}
                </p>
              </div>
            )}

            {/* Empty State */}
            {messages.length === 0 && !agentStatus?.agentInfo?.instructions && (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mb-4 dark:bg-blue-900/50">
                  <MessageCircle className="h-8 w-8 text-blue-600 dark:text-blue-400" />
                </div>
                <h3 className="text-lg font-medium text-stone-900 dark:text-stone-100 mb-2">
                  Start a conversation
                </h3>
                <p className="text-stone-600 dark:text-stone-400 text-sm max-w-md">
                  Send a message to begin chatting with {agentDisplayName || agentName}.
                  This agent will respond using its configured models and capabilities.
                </p>
              </div>
            )}

            {/* Error State */}
            {error && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg dark:bg-red-900/20 dark:border-red-800">
                <p className="text-sm text-red-800 dark:text-red-200">
                  {error}
                </p>
              </div>
            )}

            {/* Messages */}
            {messages.map((message) => (
              <ChatBubble 
                key={message.id} 
                message={message}
              />
            ))}

            {/* Typing Indicator */}
            {showTyping && (
              <TypingIndicator agentName={agentDisplayName || agentName} />
            )}

            <div ref={messagesEndRef} />
          </ScrollArea>

          {/* Message Input */}
          <ChatMessageInput
            onSendMessage={handleSendMessage}
            isLoading={isLoading}
            disabled={!isConnected}
            placeholder={
              isConnected 
                ? `Message ${agentDisplayName || agentName}...`
                : "Agent is not available for chat"
            }
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}