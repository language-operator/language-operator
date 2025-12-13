import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { z } from 'zod'

const updateToolSchema = z.object({
  type: z.string().optional(),
  description: z.string().optional(),
  spec: z.object({
    type: z.string().optional(),
    image: z.string().optional(),
    endpoint: z.string().url().optional(),
    port: z.number().int().min(1).max(65535).optional(),
    healthCheckPath: z.string().optional(),
    environment: z.record(z.string(), z.string()).optional(),
    resources: z.object({
      requests: z.object({
        cpu: z.string().optional(),
        memory: z.string().optional(),
      }).optional(),
      limits: z.object({
        cpu: z.string().optional(),
        memory: z.string().optional(),
      }).optional(),
    }).optional(),
    enabled: z.boolean().optional(),
    requireApproval: z.boolean().optional(),
    timeout: z.number().int().min(1).optional(),
    retries: z.number().int().min(0).optional(),
  }).optional()
})

// GET /api/tools/[name] - Get a specific tool
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
    const tool = await k8sClient.getLanguageTool(name, namespace)
    
    if (!tool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }
    
    return NextResponse.json({ tool })
  } catch (error) {
    console.error('Error fetching tool:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

// PATCH /api/tools/[name] - Update a specific tool
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
    const validatedData = updateToolSchema.parse(body)
    
    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    const namespace = organization.namespace

    // Get existing tool
    const existingTool = await k8sClient.getLanguageTool(name, namespace)
    if (!existingTool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    // Update the tool
    const updatedTool = await k8sClient.updateLanguageTool(name, namespace, {
      metadata: {
        ...existingTool.metadata,
        annotations: {
          ...existingTool.metadata.annotations,
          'langop.io/updated-at': new Date().toISOString(),
          'langop.io/updated-by': session.user.email || 'unknown'
        }
      },
      spec: {
        ...existingTool.spec,
        ...validatedData.spec,
        type: validatedData.type || validatedData.spec?.type || existingTool.spec.type,
      }
    })

    // Log the update for audit trail
    console.log(`Tool updated: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ tool: updatedTool })
  } catch (error) {
    console.error('Error updating tool:', error)
    
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

// DELETE /api/tools/[name] - Delete a specific tool
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

    // Check if tool exists
    const existingTool = await k8sClient.getLanguageTool(name, namespace)
    if (!existingTool) {
      return NextResponse.json({ error: 'Tool not found' }, { status: 404 })
    }

    // Delete the tool
    await k8sClient.deleteLanguageTool(name, namespace)

    // Log the deletion for audit trail
    console.log(`Tool deleted: ${name} by ${session.user.email} in ${namespace}`)

    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting tool:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}