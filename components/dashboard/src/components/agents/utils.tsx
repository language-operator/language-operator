import { AlertCircle, CheckCircle, Clock } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'

export function formatTimeAgo(timestamp?: string | Date) {
  if (!timestamp) return 'Unknown'
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (days > 0) return `${days} day${days !== 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours !== 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`
  return 'Just now'
}

// Simple ANSI code converter
export function convertAnsiToHtml(text: string): string {
  if (!text) return text;

  // Basic ANSI color mappings
  const ansiToClass: { [key: string]: string } = {
    '\x1b[30m': '<span class="ansi-black-fg">',      // Black
    '\x1b[31m': '<span class="ansi-red-fg">',        // Red
    '\x1b[32m': '<span class="ansi-green-fg">',      // Green
    '\x1b[33m': '<span class="ansi-yellow-fg">',     // Yellow
    '\x1b[34m': '<span class="ansi-blue-fg">',       // Blue
    '\x1b[35m': '<span class="ansi-magenta-fg">',    // Magenta
    '\x1b[36m': '<span class="ansi-cyan-fg">',       // Cyan
    '\x1b[37m': '<span class="ansi-white-fg">',      // White
    '\x1b[90m': '<span class="ansi-bright-black-fg">', // Bright Black (Gray)
    '\x1b[91m': '<span class="ansi-bright-red-fg">',   // Bright Red
    '\x1b[92m': '<span class="ansi-bright-green-fg">', // Bright Green
    '\x1b[93m': '<span class="ansi-bright-yellow-fg">', // Bright Yellow
    '\x1b[94m': '<span class="ansi-bright-blue-fg">',  // Bright Blue
    '\x1b[95m': '<span class="ansi-bright-magenta-fg">', // Bright Magenta
    '\x1b[96m': '<span class="ansi-bright-cyan-fg">',  // Bright Cyan
    '\x1b[97m': '<span class="ansi-bright-white-fg">', // Bright White
    '\x1b[1;36m': '<span class="ansi-bold ansi-cyan-fg">', // Bold Cyan
    '\x1b[1m': '<span class="ansi-bold">',             // Bold
    '\x1b[0m': '</span>',                              // Reset
  };

  let result = text;

  // Replace ANSI codes with HTML spans
  Object.entries(ansiToClass).forEach(([code, replacement]) => {
    result = result.split(code).join(replacement);
  });

  return result;
}

export function getStatusIcon(agent: LanguageAgent) {
  const phase = agent.status?.phase || 'Unknown'

  if (phase === 'Running') {
    return <CheckCircle className="h-5 w-5 text-green-500" />
  } else if (phase === 'Pending') {
    return <Clock className="h-5 w-5 text-yellow-500" />
  } else if (phase === 'Failed') {
    return <AlertCircle className="h-5 w-5 text-red-500" />
  } else if (phase === 'Succeeded') {
    return <CheckCircle className="h-5 w-5 text-blue-500" />
  } else {
    return <AlertCircle className="h-5 w-5 text-gray-500" />
  }
}

export function getStatusColor(agent: LanguageAgent) {
  const phase = agent.status?.phase || 'Unknown'

  if (phase === 'Running') {
    return 'text-green-600'
  } else if (phase === 'Pending') {
    return 'text-yellow-600'
  } else if (phase === 'Failed') {
    return 'text-red-600'
  } else if (phase === 'Succeeded') {
    return 'text-blue-600'
  } else {
    return 'text-gray-600'
  }
}
