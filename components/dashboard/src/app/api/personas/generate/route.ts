import { NextRequest, NextResponse } from 'next/server'
import { getUserOrganization } from '@/lib/organization-context'
import { requirePermission } from '@/lib/permissions'
import { 
  createErrorResponse, 
  createAuthenticationRequiredError, 
  createPermissionDeniedError,
  ModelNotAvailableError,
  ModelEndpointError,
  ModelResponseError,
  GenerationParsingError,
  GenerationTimeoutError
} from '@/lib/api-error-handler'
import { k8sClient } from '@/lib/k8s-client'
import { serviceResolver } from '@/lib/service-resolver'

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

    // Fetch the LanguageModel resource to get the actual model name
    let actualModelName: string
    let modelStatus: string | undefined
    try {
      const modelResource = await k8sClient.getLanguageModel(organization.namespace, modelName)
      const modelBody = (modelResource as any)?.body || modelResource
      actualModelName = modelBody.spec?.modelName
      modelStatus = modelBody.status?.phase

      if (!actualModelName) {
        throw new ModelNotAvailableError(modelName, organization.namespace, 'missing_spec')
      }

      // Check if model is ready
      if (modelStatus && modelStatus !== 'Ready') {
        throw new ModelNotAvailableError(modelName, organization.namespace, modelStatus)
      }

      console.log(`Resolved model name: ${modelName} -> ${actualModelName} (status: ${modelStatus})`)
    } catch (error: any) {
      console.error('Failed to fetch LanguageModel:', error)
      
      // Check if it's a 404 error (model not found)
      if (error?.response?.statusCode === 404 || error?.statusCode === 404) {
        throw new ModelNotAvailableError(modelName, organization.namespace, 'not_found')
      }
      
      // Re-throw our custom errors
      if (error instanceof ModelNotAvailableError) {
        throw error
      }
      
      // For other errors, wrap in a generic model availability error
      throw new ModelNotAvailableError(modelName, organization.namespace, 'fetch_failed')
    }

    // Resolve model endpoint based on environment (K8s vs Docker Compose)  
    const modelEndpoint = serviceResolver.resolveModelUrl(modelName, organization.namespace, 8000)
    
    console.log(`Environment: ${JSON.stringify(serviceResolver.getEnvironmentInfo())}`)
    console.log(`Resolved model endpoint: ${modelEndpoint}`)

    // Set a timeout for the generation request
    const timeoutSeconds = 60
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeoutSeconds * 1000)

    let response: Response
    try {
      response = await fetch(modelEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          model: actualModelName, // LiteLLM expects the actual model name from spec.modelName
          messages: [
            { role: 'system', content: systemPrompt },
            { role: 'user', content: userPrompt }
          ],
          max_tokens: 2048,
          temperature: 0.7,
        }),
        signal: controller.signal
      })
    } catch (error: any) {
      clearTimeout(timeoutId)
      console.error('Model endpoint error:', error)
      
      if (error.name === 'AbortError') {
        throw new GenerationTimeoutError(modelName, timeoutSeconds)
      }
      
      // Map different network errors to specific error types
      const errorMessage = error.message || ''
      throw new ModelEndpointError(modelName, modelEndpoint, errorMessage)
    }

    clearTimeout(timeoutId)

    if (!response.ok) {
      const errorData = await response.text()
      console.error('Model API error:', response.status, errorData)
      throw new ModelResponseError(modelName, response.status, errorData)
    }

    let data: any
    try {
      data = await response.json()
    } catch (error) {
      console.error('Failed to parse model response as JSON:', error)
      const responseText = await response.text()
      throw new ModelResponseError(modelName, response.status, `Invalid JSON response: ${responseText.slice(0, 200)}`)
    }

    const generatedText = data.choices?.[0]?.message?.content
    if (!generatedText) {
      console.error('No generated text in model response:', data)
      throw new ModelResponseError(modelName, response.status, 'Model response missing generated text')
    }

    console.log('Generated text:', generatedText)

    // Parse the generated JSON
    let personaData
    try {
      // Try to extract JSON from markdown code blocks if present
      const jsonMatch = generatedText.match(/```(?:json)?\s*([\s\S]*?)\s*```/)
      const jsonText = jsonMatch ? jsonMatch[1] : generatedText
      personaData = JSON.parse(jsonText.trim())
      
      // Validate that required fields are present
      if (!personaData.displayName || !personaData.systemPrompt) {
        throw new GenerationParsingError(generatedText)
      }
    } catch (parseError) {
      console.error('Failed to parse generated JSON:', parseError, generatedText)
      if (parseError instanceof GenerationParsingError) {
        throw parseError
      }
      throw new GenerationParsingError(generatedText)
    }

    console.log('Parsed persona data:', personaData)

    return NextResponse.json({
      success: true,
      data: personaData,
    })

  } catch (error: any) {
    console.error('Error generating persona:', error)
    
    // Our custom errors already have detailed messages, pass them through
    if (error instanceof ModelNotAvailableError ||
        error instanceof ModelEndpointError ||
        error instanceof ModelResponseError ||
        error instanceof GenerationParsingError ||
        error instanceof GenerationTimeoutError) {
      return createErrorResponse(error)
    }
    
    // For unexpected errors, provide a generic fallback
    return createErrorResponse(error, 'Failed to generate persona due to an unexpected error')
  }
}
