interface LastConversation {
  conversationId: string
  agentName: string
  clusterName: string
  timestamp: number
}

const STORAGE_KEY_PREFIX = 'lastConversation'

function getStorageKey(userId?: string): string {
  return userId ? `${STORAGE_KEY_PREFIX}_${userId}` : STORAGE_KEY_PREFIX
}

export function saveLastConversation(
  conversationId: string,
  agentName: string,
  clusterName: string,
  userId?: string
): void {
  if (typeof window === 'undefined') return

  try {
    const data: LastConversation = {
      conversationId,
      agentName,
      clusterName,
      timestamp: Date.now()
    }
    
    const key = getStorageKey(userId)
    localStorage.setItem(key, JSON.stringify(data))
  } catch (error) {
    console.error('Failed to save last conversation:', error)
  }
}

export function getLastConversation(userId?: string): LastConversation | null {
  if (typeof window === 'undefined') return null

  try {
    const key = getStorageKey(userId)
    const stored = localStorage.getItem(key)
    
    if (!stored) return null
    
    const data = JSON.parse(stored) as LastConversation
    
    const ONE_WEEK_MS = 7 * 24 * 60 * 60 * 1000
    if (Date.now() - data.timestamp > ONE_WEEK_MS) {
      clearLastConversation(userId)
      return null
    }
    
    return data
  } catch (error) {
    console.error('Failed to get last conversation:', error)
    return null
  }
}

export function clearLastConversation(userId?: string): void {
  if (typeof window === 'undefined') return

  try {
    const key = getStorageKey(userId)
    localStorage.removeItem(key)
  } catch (error) {
    console.error('Failed to clear last conversation:', error)
  }
}

export function clearAllConversationStorage(): void {
  if (typeof window === 'undefined') return

  try {
    const keys = Object.keys(localStorage)
    keys.forEach(key => {
      if (key.startsWith(STORAGE_KEY_PREFIX)) {
        localStorage.removeItem(key)
      }
    })
  } catch (error) {
    console.error('Failed to clear all conversation storage:', error)
  }
}