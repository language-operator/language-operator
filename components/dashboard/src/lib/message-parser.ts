// Utility functions for parsing agent messages and separating thinking content from responses

export interface ParsedMessage {
  thinkingContent: string[]
  responseContent: string
  hasThinking: boolean
}

/**
 * Parses agent response content to separate [THINK] sections from actual response
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

  // Extract all [THINK] sections using a more compatible approach
  const thinkingContent: string[] = []
  const sections = content.split('[THINK]')
  
  // Skip first section if it's before any [THINK] tag, collect the rest
  let remainingContent = sections[0] // Content before first [THINK]
  
  for (let i = 1; i < sections.length; i++) {
    const section = sections[i].trim()
    if (section) {
      thinkingContent.push(section)
    }
  }
  
  // If there were [THINK] sections, try to find remaining content after them
  if (thinkingContent.length > 0) {
    // Look for content that comes after all thinking sections
    const lastSection = sections[sections.length - 1]
    // Try to extract JSON or clean text from the last section
    remainingContent = lastSection || remainingContent
  }

  // Try to parse any remaining content as JSON to extract the actual message
  let responseContent = remainingContent

  // Check if remaining content contains JSON with a message field
  try {
    const jsonMatch = remainingContent.match(/\{.*"message"\s*:\s*"([^"]*)".*\}/)
    if (jsonMatch) {
      // Try to parse the full JSON
      const jsonStr = remainingContent.match(/\{.*\}/)?.[0]
      if (jsonStr) {
        const parsed = JSON.parse(jsonStr)
        if (parsed.message) {
          responseContent = parsed.message
        }
      }
    }
  } catch (error) {
    // If JSON parsing fails, keep the remaining content as-is
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
 * @returns True if the content contains [THINK] tags
 */
export function hasThinkingContent(content: string): boolean {
  return /\[THINK\]/.test(content)
}