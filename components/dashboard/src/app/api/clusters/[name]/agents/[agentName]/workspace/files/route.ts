import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { workspaceClient } from '@/lib/workspace-client'
import { WorkspaceError } from '@/types/workspace'

// GET /api/clusters/[name]/agents/[agentName]/workspace/files?path=/ - List directory contents
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    const { searchParams } = new URL(request.url)
    const path = searchParams.get('path') || '/'

    try {
      const result = await workspaceClient.listFiles(
        organization.namespace, 
        agentName, 
        path
      )
      
      return NextResponse.json(result)
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        const statusCode = getStatusCodeForWorkspaceError(error.code)
        return NextResponse.json({ 
          error: error.message, 
          code: error.code,
          details: error.details 
        }, { status: statusCode })
      }
      
      console.error('Workspace list error:', error)
      return NextResponse.json({ error: 'Failed to list workspace files' }, { status: 500 })
    }
  } catch (error: any) {
    console.error('Workspace API error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

// POST /api/clusters/[name]/agents/[agentName]/workspace/files?path=/ - Upload file
export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions for file upload' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    
    try {
      const formData = await request.formData()
      const file = formData.get('file') as File
      const path = formData.get('path') as string || '/'
      
      if (!file) {
        return NextResponse.json({ error: 'No file provided' }, { status: 400 })
      }

      const fileBuffer = Buffer.from(await file.arrayBuffer())
      const filePath = path.endsWith('/') ? `${path}${file.name}` : `${path}/${file.name}`

      await workspaceClient.uploadFile(
        organization.namespace,
        agentName,
        path,
        fileBuffer,
        file.name
      )
      
      return NextResponse.json({ 
        message: 'File uploaded successfully',
        filename: file.name,
        path: filePath,
        size: fileBuffer.length
      })
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        const statusCode = getStatusCodeForWorkspaceError(error.code)
        return NextResponse.json({ 
          error: error.message, 
          code: error.code,
          details: error.details 
        }, { status: statusCode })
      }
      
      console.error('Workspace upload error:', error)
      return NextResponse.json({ error: 'Failed to upload file' }, { status: 500 })
    }
  } catch (error: any) {
    console.error('Workspace upload API error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

// DELETE /api/clusters/[name]/agents/[agentName]/workspace/files?path=/file.txt - Delete file
export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; agentName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'edit')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions for file deletion' }, { status: 403 })
    }

    const { name: clusterName, agentName } = await params
    const { searchParams } = new URL(request.url)
    const path = searchParams.get('path')

    if (!path) {
      return NextResponse.json({ error: 'File path is required' }, { status: 400 })
    }

    try {
      await workspaceClient.deleteFile(
        organization.namespace,
        agentName,
        path
      )
      
      return NextResponse.json({ 
        message: 'File deleted successfully',
        path 
      })
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        const statusCode = getStatusCodeForWorkspaceError(error.code)
        return NextResponse.json({ 
          error: error.message, 
          code: error.code,
          details: error.details 
        }, { status: statusCode })
      }
      
      console.error('Workspace delete error:', error)
      return NextResponse.json({ error: 'Failed to delete file' }, { status: 500 })
    }
  } catch (error: any) {
    console.error('Workspace delete API error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

// Helper function to map workspace error codes to HTTP status codes
function getStatusCodeForWorkspaceError(code: WorkspaceError['code']): number {
  switch (code) {
    case 'NOT_FOUND':
      return 404
    case 'ACCESS_DENIED':
      return 403
    case 'SIZE_LIMIT':
      return 413 // Payload Too Large
    case 'INVALID_PATH':
      return 400
    case 'TIMEOUT':
      return 408 // Request Timeout
    case 'POD_ERROR':
    default:
      return 500
  }
}