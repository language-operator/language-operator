import { NextRequest, NextResponse } from 'next/server'
import { fetchToolCatalog, getToolById, transformCatalogEntryToLanguageTool } from '@/lib/tool-catalog'
import { k8sClient } from '@/lib/k8s-client'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session?.activeOrganization?.namespace) {
      return NextResponse.json(
        { error: 'No active organization found' },
        { status: 401 }
      )
    }

    const body = await request.json()
    const { toolId, clusterName } = body

    if (!toolId) {
      return NextResponse.json(
        { error: 'Missing required field: toolId' },
        { status: 400 }
      )
    }

    // Use the user's organization namespace
    const namespace = session.activeOrganization.namespace

    // Fetch the tool from catalog
    const catalog = await fetchToolCatalog()
    const tool = getToolById(catalog, toolId)

    if (!tool) {
      return NextResponse.json(
        { error: `Tool "${toolId}" not found in catalog` },
        { status: 404 }
      )
    }

    // Transform catalog entry to LanguageTool CRD
    const languageTool = transformCatalogEntryToLanguageTool(tool, namespace, clusterName)

    try {
      // Apply the LanguageTool CRD to Kubernetes
      const response = await k8sClient.createLanguageTool(namespace, languageTool)

      return NextResponse.json({
        success: true,
        message: 'Tool installed successfully',
        tool: response,
      })
    } catch (k8sError: any) {
      // Check if tool already exists
      if (k8sError.response?.statusCode === 409) {
        return NextResponse.json(
          { error: 'Tool already installed' },
          { status: 409 }
        )
      }

      console.error('Kubernetes API error:', k8sError)
      return NextResponse.json(
        { 
          error: 'Failed to install tool',
          details: k8sError.body?.message || k8sError.message 
        },
        { status: 500 }
      )
    }
  } catch (error) {
    console.error('Error installing tool:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to install tool',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}