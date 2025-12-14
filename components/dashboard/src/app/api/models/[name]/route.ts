import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { z } from 'zod'

const updateModelSchema = z.object({
  provider: z.string().optional(),
  model: z.string().optional(),
  endpoint: z.string().url().optional(),
  apiKey: z.string().optional(),
  description: z.string().optional(),
  spec: z.object({
    provider: z.string().optional(),
    model: z.string().optional(),
    endpoint: z.string().url().optional(),
    apiKey: z.string().optional(),
    parameters: z.object({
      maxTokens: z.number().int().min(1).optional(),
      temperature: z.number().min(0).max(2).optional(),
      topP: z.number().min(0).max(1).optional(),
      frequencyPenalty: z.number().min(-2).max(2).optional(),
      presencePenalty: z.number().min(-2).max(2).optional(),
    }).optional(),
    contextWindow: z.number().int().min(1).optional(),
    cost: z.object({
      inputTokens: z.number().min(0).optional(),
      outputTokens: z.number().min(0).optional(),
      currency: z.string().optional()
    }).optional(),
    enabled: z.boolean().optional(),
    requireApproval: z.boolean().optional(),
  }).optional()
})

// GET /api/models/[name] - Get a specific model
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace
    const model = await k8sClient.getLanguageModel(namespace, name)
    
    if (!model) {
      return NextResponse.json({ error: 'Model not found' }, { status: 404 })
    }
    
    return NextResponse.json({ data: model })
  } catch (error) {
    console.error('Error fetching model:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

// PATCH /api/models/[name] - Update a specific model
export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = updateModelSchema.parse(body)
    
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Get existing model
    const existingModel = await k8sClient.getLanguageModel(namespace, name)
    if (!existingModel) {
      return NextResponse.json({ error: 'Model not found' }, { status: 404 })
    }

    // Update the model
    const updatedModel = await k8sClient.updateLanguageModel(namespace, name, {
      metadata: {
        ...existingModel.metadata,
        annotations: {
          ...existingModel.metadata.annotations,
          'langop.io/updated-at': new Date().toISOString(),
          'langop.io/updated-by': session.user.email || 'unknown'
        }
      },
      spec: {
        ...existingModel.spec,
        ...validatedData.spec,
        provider: validatedData.provider || validatedData.spec?.provider || existingModel.spec.provider,
        model: validatedData.model || validatedData.spec?.model || existingModel.spec.model,
        endpoint: validatedData.endpoint || validatedData.spec?.endpoint || existingModel.spec.endpoint,
        ...(validatedData.apiKey !== undefined && { apiKey: validatedData.apiKey }),
        ...(validatedData.spec?.apiKey !== undefined && { apiKey: validatedData.spec.apiKey }),
      }
    })

    // Log the update for audit trail
    console.log(`Model updated: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ data: updatedModel })
  } catch (error) {
    console.error('Error updating model:', error)
    
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: 'Invalid input', details: error.issues },
        { status: 400 }
      )
    }

    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

// PUT /api/models/[name] - Update a specific model (full replace)
export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    console.log('🔥 PUT /api/models/[name] - Starting model update')
    const { name } = await params
    console.log('🔥 Model name:', name)
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const body = await request.json()
    const validatedData = updateModelSchema.parse(body)
    
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Get existing model
    const existingModel = await k8sClient.getLanguageModel(namespace, name)
    if (!existingModel) {
      return NextResponse.json({ error: 'Model not found' }, { status: 404 })
    }

    console.log('🔥 About to call replaceLanguageModel with namespace:', namespace, 'name:', name)
    // Replace the model using PUT semantics
    const updatedModel = await k8sClient.replaceLanguageModel(namespace, name, {
      metadata: {
        ...existingModel.metadata,
        annotations: {
          ...existingModel.metadata.annotations,
          'langop.io/updated-at': new Date().toISOString(),
          'langop.io/updated-by': session.user.email || 'unknown'
        }
      },
      spec: {
        ...existingModel.spec,
        ...validatedData.spec,
        provider: validatedData.provider || validatedData.spec?.provider || existingModel.spec.provider,
        model: validatedData.model || validatedData.spec?.model || existingModel.spec.model,
        endpoint: validatedData.endpoint || validatedData.spec?.endpoint || existingModel.spec.endpoint,
        ...(validatedData.apiKey !== undefined && { apiKey: validatedData.apiKey }),
        ...(validatedData.spec?.apiKey !== undefined && { apiKey: validatedData.spec.apiKey }),
      }
    })

    // Log the update for audit trail
    console.log(`Model updated via PUT: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ data: updatedModel })
  } catch (error) {
    console.error('🔥 Error updating model via PUT:', error)
    console.error('🔥 Error stack:', error instanceof Error ? error.stack : 'No stack trace')
    
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: 'Invalid input', details: error.issues },
        { status: 400 }
      )
    }

    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

// DELETE /api/models/[name] - Delete a specific model
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized - no organization' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Check if model exists
    const existingModel = await k8sClient.getLanguageModel(namespace, name)
    if (!existingModel) {
      return NextResponse.json({ error: 'Model not found' }, { status: 404 })
    }

    // Delete the model
    await k8sClient.deleteLanguageModel(namespace, name)

    // Log the deletion for audit trail
    console.log(`Model deleted: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting model:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}