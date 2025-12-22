import { NextRequest, NextResponse } from 'next/server'
import { getUserOrganization } from '@/lib/organization-context'
import { requirePermission } from '@/lib/permissions'
import { createErrorResponse, createAuthenticationRequiredError, createPermissionDeniedError } from '@/lib/api-error-handler'

// POST /api/personas/generate - Generate a persona using a cluster model
export async function POST(request: NextRequest) {
  try {
    // Get user's selected organization
    const { user, organization, userRole } = await getUserOrganization(request)

    if (!user?.id) {
      throw createAuthenticationRequiredError()
    }

    // Check permissions
    const hasPermission = await requirePermission(user.id, organization.id, 'create')
    if (!hasPermission) {
      throw createPermissionDeniedError('generate personas', 'AI-generated personas', userRole)
    }

    const body = await request.json()
    const { idea, modelName } = body

    if (!idea) {
      return createErrorResponse(
        new Error('Idea is required'),
        'Invalid request data'
      )
    }

    if (!modelName) {
      return createErrorResponse(
        new Error('Model name is required'),
        'Invalid request data'
      )
    }

    // Generate persona using the cluster model via LiteLLM proxy
    // The model is exposed as a k8s service at: http://{modelName}.{namespace}.svc.cluster.local
    const systemPrompt = `You are an expert AI persona designer. Your task is to create detailed, well-structured personas for AI agents based on user ideas.

When given an idea, you must respond with ONLY a valid JSON object (no markdown, no explanation) with this exact structure:
{
  "displayName": "A clear, descriptive name for this persona",
  "systemPrompt": "Core behavioral instructions that define how this persona acts - be specific and detailed",
  "tone": "professional|friendly|casual|formal|empathetic",
  "language": "The primary language (e.g., English, Spanish, etc.)",
  "instructions": [
    "Specific instruction 1",
    "Specific instruction 2",
    "Specific instruction 3"
  ]
}

Guidelines:
- Make systemPrompt comprehensive and specific to the use case
- Choose an appropriate tone that matches the persona's purpose
- Add 2-5 instructions that provide additional behavioral guidance
- Instructions can be formatted in markdown for better readability
- Make the persona practical and immediately useful`

    const userPrompt = `Create a persona for: ${idea}`

    console.log(`Generating persona with model ${modelName} for idea: "${idea}"`)

    // Call the model via LiteLLM proxy (OpenAI-compatible API)
    // The service is at: http://{modelName}.{namespace}.svc.cluster.local/v1/chat/completions
    const modelEndpoint = `http://${modelName}.${organization.namespace}.svc.cluster.local/v1/chat/completions`

    console.log(`Calling model endpoint: ${modelEndpoint}`)

    const response = await fetch(modelEndpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        model: modelName, // LiteLLM expects the model name
        messages: [
          { role: 'system', content: systemPrompt },
          { role: 'user', content: userPrompt }
        ],
        max_tokens: 2048,
        temperature: 0.7,
      })
    })

    if (!response.ok) {
      const errorData = await response.text()
      console.error('Model API error:', errorData)
      throw new Error(`Failed to generate persona: ${response.statusText}`)
    }

    const data = await response.json()
    const generatedText = data.choices[0].message.content

    console.log('Generated text:', generatedText)

    // Parse the generated JSON
    let personaData
    try {
      // Try to extract JSON from markdown code blocks if present
      const jsonMatch = generatedText.match(/```(?:json)?\s*([\s\S]*?)\s*```/)
      const jsonText = jsonMatch ? jsonMatch[1] : generatedText
      personaData = JSON.parse(jsonText.trim())
    } catch (parseError) {
      console.error('Failed to parse generated JSON:', generatedText)
      throw new Error('Generated response was not valid JSON')
    }

    console.log('Parsed persona data:', personaData)

    return NextResponse.json({
      success: true,
      data: personaData,
    })

  } catch (error: any) {
    console.error('Error generating persona:', error)
    return createErrorResponse(error, 'Failed to generate persona')
  }
}
