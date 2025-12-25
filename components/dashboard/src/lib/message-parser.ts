// Utility functions for parsing agent messages and separating thinking content from responses

export interface ParsedMessage {
  thinkingContent: string[]
  responseContent: string
  hasThinking: boolean
}

/**
 * Parses agent response content to separate [THINK]...[/THINK] sections from actual response
 * @param content - Raw agent response content
 * @returns Parsed message with thinking and response content separated
 */
export function parseAgentMessage(content: string): ParsedMessage {
  if (!content) {
    return {
      thinkingContent: [],
      responseContent: '',
      hasThinking: false
    }
  }

  // Extract all [THINK]...[/THINK] sections using regex with dotall flag
  const thinkingContent: string[] = []
  const thinkRegex = /\[THINK\]([\s\S]*?)\[\/THINK\]/g
  let remainingContent = content
  
  // Extract thinking content and remove from remaining content
  let match
  while ((match = thinkRegex.exec(content)) !== null) {
    const thinkContent = match[1].trim()
    if (thinkContent) {
      thinkingContent.push(thinkContent)
    }
    // Remove this [THINK]...[/THINK] section from remaining content
    remainingContent = remainingContent.replace(match[0], '').trim()
  }

  // Try to parse any remaining content as JSON to extract the actual message
  let responseContent = remainingContent

  // Check if remaining content contains JSON with a message field
  try {
    // Try to parse the entire remaining content as JSON first
    const jsonStr = remainingContent.match(/\{[\s\S]*\}/)?.[0]
    if (jsonStr) {
      const parsed = JSON.parse(jsonStr)
      if (parsed.message) {
        responseContent = parsed.message
      }
    }
  } catch (error) {
    // If full JSON parsing fails, try to extract message field with regex
    try {
      const jsonMatch = remainingContent.match(/"message"\s*:\s*"([^"]*)"/)
      if (jsonMatch) {
        responseContent = jsonMatch[1]
      }
    } catch (e) {
      // If all parsing fails, keep the remaining content as-is
    }
  }

  // If no clean response found, try to extract any non-[THINK] text
  if (!responseContent && thinkingContent.length > 0) {
    responseContent = 'Response content not clearly separated from thinking.'
  }

  return {
    thinkingContent,
    responseContent: responseContent || content, // Fallback to original content
    hasThinking: thinkingContent.length > 0
  }
}

/**
 * Formats thinking content for display
 * @param thinkingContent - Array of thinking sections
 * @returns Formatted thinking content string
 */
export function formatThinkingContent(thinkingContent: string[]): string {
  return thinkingContent
    .map((section, index) => `${index + 1}. ${section}`)
    .join('\n\n')
}

/**
 * Checks if a message contains thinking content
 * @param content - Message content to check
 * @returns True if the content contains [THINK]...[/THINK] tags
 */
export function hasThinkingContent(content: string): boolean {
  return /\[THINK\][\s\S]*?\[\/THINK\]/.test(content)
}