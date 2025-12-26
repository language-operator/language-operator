import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { workspaceClient } from '@/lib/workspace-client'
import { WorkspaceError, VIEWABLE_FILE_EXTENSIONS } from '@/types/workspace'

// GET /api/clusters/[name]/agents/[agentName]/workspace/files/view?path=/file.txt - Get file content for viewing
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

    // Check if file type is viewable
    const filename = path.split('/').pop() || ''
    const extension = filename.toLowerCase().split('.').pop()
    const isViewable = VIEWABLE_FILE_EXTENSIONS.some(ext => 
      ext.substring(1) === extension || filename.toLowerCase().endsWith(ext)
    )

    if (!isViewable) {
      return NextResponse.json({ 
        error: 'File type not supported for viewing',
        code: 'UNSUPPORTED_TYPE',
        supportedTypes: VIEWABLE_FILE_EXTENSIONS 
      }, { status: 415 })
    }

    try {
      const content = await workspaceClient.viewFile(
        organization.namespace,
        agentName,
        path
      )
      
      // Detect if content is binary (contains null bytes or non-printable characters)
      const isBinary = content.includes('\0') || 
        /[\x00-\x08\x0E-\x1F\x7F-\xFF]/.test(content.slice(0, 512))
      
      if (isBinary) {
        return NextResponse.json({ 
          error: 'File appears to be binary and cannot be viewed as text',
          code: 'BINARY_FILE'
        }, { status: 415 })
      }

      // Determine syntax highlighting language based on file extension
      const language = getLanguageFromExtension(extension)
      const mimeType = getMimeTypeFromExtension(extension)
      
      return NextResponse.json({
        content,
        encoding: 'utf8',
        size: Buffer.byteLength(content, 'utf8'),
        filename,
        path,
        language,
        mimeType,
        isViewable: true,
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
      
      console.error('Workspace view error:', error)
      return NextResponse.json({ error: 'Failed to view file' }, { status: 500 })
    }
  } catch (error: any) {
    console.error('Workspace view API error:', error)
    return NextResponse.json({ error: 'Internal server error' }, { status: 500 })
  }
}

// Helper function to determine syntax highlighting language
function getLanguageFromExtension(extension?: string): string {
  if (!extension) return 'text'
  
  const languageMap: Record<string, string> = {
    'js': 'javascript',
    'jsx': 'javascript',
    'ts': 'typescript',
    'tsx': 'typescript',
    'py': 'python',
    'rb': 'ruby',
    'go': 'go',
    'rs': 'rust',
    'java': 'java',
    'c': 'c',
    'cpp': 'cpp',
    'cc': 'cpp',
    'cxx': 'cpp',
    'h': 'c',
    'hpp': 'cpp',
    'hh': 'cpp',
    'hxx': 'cpp',
    'cs': 'csharp',
    'php': 'php',
    'sh': 'bash',
    'bash': 'bash',
    'zsh': 'bash',
    'fish': 'bash',
    'sql': 'sql',
    'html': 'html',
    'htm': 'html',
    'xml': 'xml',
    'css': 'css',
    'scss': 'scss',
    'sass': 'sass',
    'less': 'less',
    'json': 'json',
    'yaml': 'yaml',
    'yml': 'yaml',
    'toml': 'toml',
    'ini': 'ini',
    'conf': 'ini',
    'cfg': 'ini',
    'md': 'markdown',
    'markdown': 'markdown',
    'txt': 'text',
    'log': 'text',
    'csv': 'csv',
    'dockerfile': 'dockerfile',
    'makefile': 'makefile',
    'gitignore': 'gitignore',
    'bas': 'basic',
    'basic': 'basic',
    'qb': 'basic',
    'vb': 'vbnet',
  }
  
  return languageMap[extension.toLowerCase()] || 'text'
}

// Helper function to determine MIME type
function getMimeTypeFromExtension(extension?: string): string {
  if (!extension) return 'text/plain'
  
  const mimeTypeMap: Record<string, string> = {
    'js': 'text/javascript',
    'jsx': 'text/javascript',
    'ts': 'text/typescript',
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
    'html': 'text/html',
    'xml': 'application/xml',
    'css': 'text/css',
    'json': 'application/json',
    'yaml': 'text/yaml',
    'yml': 'text/yaml',
    'md': 'text/markdown',
    'txt': 'text/plain',
    'csv': 'text/csv',
    'sql': 'text/x-sql',
    'sh': 'text/x-shellscript',
    'bash': 'text/x-shellscript',
    'bas': 'text/x-basic',
    'basic': 'text/x-basic',
    'qb': 'text/x-basic',
    'vb': 'text/vb',
  }
  
  return mimeTypeMap[extension.toLowerCase()] || 'text/plain'
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