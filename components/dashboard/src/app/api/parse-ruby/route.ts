import { NextRequest, NextResponse } from 'next/server'

interface ParsedTask {
  id: string
  name: string
  type: 'symbolic' | 'neural'
  instructions: string
  inputs: Array<{name: string, type: string}>
  outputs: Array<{name: string, type: string}>
  codeBlock: string
}

export async function POST(request: NextRequest) {
  try {
    const { code } = await request.json()
    
    if (!code || typeof code !== 'string') {
      return NextResponse.json(
        { success: false, error: 'Code is required' },
        { status: 400 }
      )
    }

    const tasks: ParsedTask[] = []

    // Use improved regex parsing that handles missing end blocks properly
    // Only match task definitions (task(...)), not execute_task calls
    const taskRegex = /(?<!execute_)task\s*\(\s*:(\w+)\s*,\s*([\s\S]*?)\)\s*(?:do\s*(\|[^|]*\|)?\s*([\s\S]*?)\s*end)?(?=\s*(?:\n\s*(?:task\s*\(|end\s|main\s|constraints\s|output\s|agent\s|\s*$)))/gm
    
    let match
    let taskIndex = 1
    
    while ((match = taskRegex.exec(code)) !== null) {
      const [fullMatch, taskName, paramsStr, doParams, doBlock] = match
      
      // Parse parameters from the string
      let instructions = ''
      const inputs: Array<{name: string, type: string}> = []
      const outputs: Array<{name: string, type: string}> = []
      
      if (paramsStr) {
        // Extract instructions - handle both double and single quoted strings with proper escaping
        let instructionsMatch = paramsStr.match(/instructions:\s*"((?:[^"\\]|\\.*)*)"/)
        if (!instructionsMatch) {
          instructionsMatch = paramsStr.match(/instructions:\s*'((?:[^'\\]|\\.*)*)'/)
        }
        if (instructionsMatch) {
          instructions = instructionsMatch[1]
        }
        
        // Extract inputs hash with key: type pairs
        const inputsMatch = paramsStr.match(/inputs:\s*\{([^}]*)\}/)
        if (inputsMatch) {
          const inputContent = inputsMatch[1]
          const inputMatches = inputContent.match(/(\w+):\s*['"](.*?)['"],?/g)
          if (inputMatches) {
            inputMatches.forEach(match => {
              const keyValueMatch = match.match(/(\w+):\s*['"](.*?)['"]/)
              if (keyValueMatch) {
                inputs.push({
                  name: keyValueMatch[1],
                  type: keyValueMatch[2]
                })
              }
            })
          }
        }
        
        // Extract outputs hash with key: type pairs  
        const outputsMatch = paramsStr.match(/outputs:\s*\{([^}]*)\}/)
        if (outputsMatch) {
          const outputContent = outputsMatch[1]
          const outputMatches = outputContent.match(/(\w+):\s*['"](.*?)['"],?/g)
          if (outputMatches) {
            outputMatches.forEach(match => {
              const keyValueMatch = match.match(/(\w+):\s*['"](.*?)['"]/)
              if (keyValueMatch) {
                outputs.push({
                  name: keyValueMatch[1],
                  type: keyValueMatch[2]
                })
              }
            })
          }
        }
      }

      // Build complete code block
      let completeCodeBlock = fullMatch.trim()
      
      // If there's a do block but it's missing 'end', add it
      if (fullMatch.includes(' do ') && doBlock && !doBlock.trim().endsWith('end') && !fullMatch.includes('\nend')) {
        completeCodeBlock += '\nend'
      }

      // Determine if task is neural based on task name and instructions
      const isNeuralTask = taskName.includes('generate') || 
                          taskName.includes('analyze') || 
                          taskName.includes('create') ||
                          instructions.toLowerCase().includes('generate') ||
                          instructions.toLowerCase().includes('create') ||
                          instructions.toLowerCase().includes('analyze') ||
                          (!doBlock || doBlock.trim().length === 0) // Tasks without implementation blocks are likely neural

      tasks.push({
        id: `task_${taskIndex++}`,
        name: taskName,
        type: isNeuralTask ? 'neural' : 'symbolic',
        instructions,
        inputs,
        outputs,
        codeBlock: completeCodeBlock
      })
    }

    // Parse main block
    let mainBlock = null
    const mainRegex = /main\s*do\s*(?:\|[^|]*\|)?\s*([\s\S]*?)\s*end/
    const mainMatch = code.match(mainRegex)
    if (mainMatch) {
      mainBlock = mainMatch[1].trim()
    }

    return NextResponse.json({
      success: true,
      data: {
        tasks,
        mainBlock
      }
    })

  } catch (error) {
    console.error('Error parsing Ruby code:', error)
    
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to parse Ruby code',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}