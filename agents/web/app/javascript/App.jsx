import { useState, useEffect } from 'react'
import ChatInterface from './components/ChatInterface'
import './App.css'

function App() {
  const [sessionId, setSessionId] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    // Create a new chat session when the app loads
    createSession()
  }, [])

  const createSession = async () => {
    try {
      setLoading(true)
      setError(null)
      
      const response = await fetch('/api/chats', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (!response.ok) {
        throw new Error('Failed to create session')
      }
      
      const data = await response.json()
      setSessionId(data.id)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleNewSession = () => {
    createSession()
  }

  if (loading) {
    return (
      <div className="app loading">
        <div className="loading-spinner"></div>
        <p>Connecting to Based MCP...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="app error">
        <h1>Error</h1>
        <p>{error}</p>
        <button onClick={createSession}>Retry</button>
      </div>
    )
  }

  return (
    <div className="app">
      <ChatInterface 
        sessionId={sessionId} 
        onNewSession={handleNewSession}
      />
    </div>
  )
}

export default App
