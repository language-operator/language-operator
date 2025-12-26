import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { workspaceClient } from '@/lib/workspace-client'
import { WorkspaceError } from '@/types/workspace'

// GET /api/clusters/[name]/agents/[agentName]/workspace/files/download?path=/file.txt - Download file
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
    const path = searchParams.get('path')

    if (!path) {
      return NextResponse.json({ error: 'File path is required' }, { status: 400 })
    }

    try {
      const fileBlob = await workspaceClient.downloadFile(
        organization.namespace,
        agentName,
        path
      )
      
      const fileBuffer = Buffer.from(await fileBlob.arrayBuffer())
      
      // Extract filename from path
      const filename = path.split('/').pop() || 'download'
      
      // Determine content type based on file extension
      const contentType = getContentType(filename)
      
      return new NextResponse(fileBuffer, {
        status: 200,
        headers: {
          'Content-Type': contentType,
          'Content-Disposition': `attachment; filename="${filename}"`,
          'Content-Length': fileBuffer.length.toString(),
        },
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
      
      console.error('Workspace download error:', error)
      return NextResponse.json({ error: 'Failed to download file' }, { status: 500 })
    }
  } catch (error: any) {
    console.error('Workspace download API error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

// Helper function to determine content type from filename
function getContentType(filename: string): string {
  const ext = filename.toLowerCase().split('.').pop()
  
  const contentTypes: Record<string, string> = {
    'txt': 'text/plain',
    'md': 'text/markdown',
    'json': 'application/json',
    'yaml': 'text/yaml',
    'yml': 'text/yaml',
    'xml': 'application/xml',
    'html': 'text/html',
    'css': 'text/css',
    'js': 'text/javascript',
    'ts': 'text/typescript',
    'jsx': 'text/javascript',
    'tsx': 'text/typescript',
    'py': 'text/x-python',
    'rb': 'text/x-ruby',
    'go': 'text/x-go',
    'rs': 'text/x-rust',
    'java': 'text/x-java',
    'c': 'text/x-c',
    'cpp': 'text/x-c++',
    'h': 'text/x-c',
    'hpp': 'text/x-c++',
    'sh': 'text/x-shellscript',
    'bash': 'text/x-shellscript',
    'zsh': 'text/x-shellscript',
    'fish': 'text/x-shellscript',
    'sql': 'text/x-sql',
    'toml': 'text/x-toml',
    'ini': 'text/plain',
    'conf': 'text/plain',
    'cfg': 'text/plain',
    'log': 'text/plain',
    'csv': 'text/csv',
    'env': 'text/plain',
    'gitignore': 'text/plain',
    'dockerignore': 'text/plain',
    'dockerfile': 'text/plain',
    'makefile': 'text/plain',
    'png': 'image/png',
    'jpg': 'image/jpeg',
    'jpeg': 'image/jpeg',
    'gif': 'image/gif',
    'svg': 'image/svg+xml',
    'pdf': 'application/pdf',
    'zip': 'application/zip',
    'tar': 'application/x-tar',
    'gz': 'application/gzip',
  }
  
  return contentTypes[ext || ''] || 'application/octet-stream'
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