import { useState, useEffect, useRef } from 'react'
import './ChatInterface.css'

function ChatInterface({ sessionId, onNewSession }) {
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [serverInfo, setServerInfo] = useState(null)
  const [expandedThinking, setExpandedThinking] = useState(new Set())
  const messagesEndRef = useRef(null)

  useEffect(() => {
    // Load session info and history
    loadSessionInfo()
    loadHistory()
  }, [sessionId])

  useEffect(() => {
    // Auto-scroll to bottom when new messages arrive
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const loadSessionInfo = async () => {
    try {
      const response = await fetch(`/api/chats/${sessionId}`)
      const data = await response.json()
      setServerInfo(data.servers)
    } catch (err) {
      console.error('Failed to load session info:', err)
    }
  }

  const loadHistory = async () => {
    try {
      const response = await fetch(`/api/chats/${sessionId}/history`)
      const data = await response.json()
      setMessages(data.messages || [])
    } catch (err) {
      console.error('Failed to load history:', err)
    }
  }

  const sendMessage = async (e) => {
    e.preventDefault()

    if (!input.trim() || sending) return

    const userMessage = {
      role: 'user',
      content: input.trim(),
      timestamp: new Date().toISOString()
    }

    setMessages(prev => [...prev, userMessage])
    setInput('')
    setSending(true)

    // Create a placeholder for the streaming assistant message
    const assistantMessageIndex = messages.length + 1
    const assistantMessage = {
      role: 'assistant',
      content: '',
      timestamp: new Date().toISOString(),
      streaming: true
    }
    setMessages(prev => [...prev, assistantMessage])

    try {
      const eventSource = new EventSource(
        `/api/chats/${sessionId}/messages?${new URLSearchParams({
          'message[content]': userMessage.content
        })}`
      )

      let accumulatedContent = ''

      // Handle heartbeat/ping events to keep connection alive
      eventSource.addEventListener('ping', (event) => {
        // Just ignore heartbeat messages - they're only to keep the connection alive
        console.debug('Received heartbeat')
      })

      eventSource.addEventListener('message', (event) => {
        const data = JSON.parse(event.data)

        if (data.type === 'chunk') {
          accumulatedContent += data.content
          setMessages(prev => {
            const newMessages = [...prev]
            newMessages[assistantMessageIndex] = {
              ...newMessages[assistantMessageIndex],
              content: accumulatedContent
            }
            return newMessages
          })
        } else if (data.type === 'complete') {
          setMessages(prev => {
            const newMessages = [...prev]
            newMessages[assistantMessageIndex] = {
              role: 'assistant',
              content: data.message.content,
              timestamp: data.message.timestamp,
              streaming: false
            }
            return newMessages
          })
          eventSource.close()
          setSending(false)
        } else if (data.type === 'error') {
          setMessages(prev => {
            const newMessages = [...prev]
            newMessages[assistantMessageIndex] = {
              role: 'error',
              content: `Error: ${data.error}`,
              timestamp: new Date().toISOString()
            }
            return newMessages
          })
          eventSource.close()
          setSending(false)
        }
      })

      eventSource.onerror = (err) => {
        console.error('EventSource error:', err)
        setMessages(prev => {
          const newMessages = [...prev]
          newMessages[assistantMessageIndex] = {
            role: 'error',
            content: 'Error: Failed to stream message',
            timestamp: new Date().toISOString()
          }
          return newMessages
        })
        eventSource.close()
        setSending(false)
      }
    } catch (err) {
      console.error('Failed to send message:', err)
      setMessages(prev => {
        const newMessages = [...prev]
        newMessages[assistantMessageIndex] = {
          role: 'error',
          content: `Error: ${err.message}`,
          timestamp: new Date().toISOString()
        }
        return newMessages
      })
      setSending(false)
    }
  }

  const clearHistory = async () => {
    if (!confirm('Clear chat history?')) return

    try {
      await fetch(`/api/chats/${sessionId}/history`, { method: 'DELETE' })
      setMessages([])
    } catch (err) {
      console.error('Failed to clear history:', err)
    }
  }

  const formatTime = (timestamp) => {
    return new Date(timestamp).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const parseMessageContent = (msg) => {
    let rawContent = ''

    // Handle different content structures
    if (typeof msg.content === 'string') {
      rawContent = msg.content
    } else if (msg.content?.content?.text) {
      // Handle nested RubyLLM response structure
      rawContent = msg.content.content.text
    } else {
      // Fallback
      rawContent = JSON.stringify(msg.content)
    }

    // Parse [THINK] blocks
    const thinkRegex = /\[THINK\]([\s\S]*?)\[\/THINK\]/g
    const thinkMatches = [...rawContent.matchAll(thinkRegex)]
    const thinking = thinkMatches.map(match => match[1].trim())
    const content = rawContent.replace(thinkRegex, '').trim()

    return { content, thinking }
  }

  const toggleThinking = (idx) => {
    setExpandedThinking(prev => {
      const newSet = new Set(prev)
      if (newSet.has(idx)) {
        newSet.delete(idx)
      } else {
        newSet.add(idx)
      }
      return newSet
    })
  }

  return (
    <div className="chat-interface">
      <header className="chat-header">
        <div className="header-left">
          <h1>Based MCP Chat</h1>
          {serverInfo && (
            <span className="server-count">
              {serverInfo.length} servers • {serverInfo.reduce((acc, s) => acc + s.tool_count, 0)} tools
            </span>
          )}
        </div>
        <div className="header-actions">
          <button onClick={clearHistory} className="btn-secondary">Clear</button>
          <button onClick={onNewSession} className="btn-secondary">New Session</button>
        </div>
      </header>

      <div className="messages-container">
        {messages.length === 0 ? (
          <div className="welcome-message">
            <h2>Welcome to Based MCP</h2>
            <p>Start a conversation with your AI assistant powered by multiple MCP servers.</p>
            {serverInfo && serverInfo.length > 0 && (
              <div className="connected-servers">
                <h3>Connected Servers:</h3>
                <ul>
                  {serverInfo.map(server => (
                    <li key={server.name}>
                      <strong>{server.name}</strong> - {server.tool_count} tools
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ) : (
          messages.map((msg, idx) => {
            const { content, thinking } = parseMessageContent(msg)
            const isExpanded = expandedThinking.has(idx)

            return (
              <div key={idx} className={`message message-${msg.role}`}>
                <div className="message-header">
                  <span className="message-role">
                    {msg.role === 'user' ? 'You' : msg.role === 'assistant' ? 'Assistant' : 'Error'}
                  </span>
                  {msg.timestamp && (
                    <span className="message-time">{formatTime(msg.timestamp)}</span>
                  )}
                </div>
                {thinking.length > 0 && (
                  <div className="thinking-section">
                    <button
                      onClick={() => toggleThinking(idx)}
                      className="thinking-toggle"
                    >
                      {isExpanded ? '▼' : '▶'} Thinking
                    </button>
                    {isExpanded && (
                      <div className="thinking-content">
                        {thinking.map((think, i) => (
                          <p key={i}>{think}</p>
                        ))}
                      </div>
                    )}
                  </div>
                )}
                <div className="message-content">
                  {content}
                </div>
              </div>
            )
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={sendMessage} className="input-form">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder={sending ? "Sending..." : "Type your message..."}
          disabled={sending}
          className="message-input"
        />
        <button 
          type="submit" 
          disabled={!input.trim() || sending}
          className="send-button"
        >
          {sending ? '...' : 'Send'}
        </button>
      </form>
    </div>
  )
}

export default ChatInterface
