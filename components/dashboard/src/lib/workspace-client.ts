import { ServiceResolver } from './service-resolver'
import { FileEntry, WorkspaceError } from '@/types/workspace'

export interface WorkspaceFile {
  name: string
  path: string
  type: 'file' | 'directory'
  size?: number
  modified?: string
  permissions?: string
}

export interface WorkspaceListResponse {
  files: WorkspaceFile[]
  path: string
}

export class WorkspaceClient {
  private serviceResolver: ServiceResolver

  constructor() {
    this.serviceResolver = new ServiceResolver()
  }

  /**
   * List files in a workspace directory
   */
  async listFiles(
    namespace: string,
    agentName: string,
    path: string = '/'
  ): Promise<WorkspaceListResponse> {
    const url = this.buildWorkspaceUrl(namespace, agentName, '/files', { path })

    try {
      const response = await fetch(url)
      
      if (!response.ok) {
        throw new WorkspaceError(
          this.mapHttpStatusToErrorCode(response.status),
          `Failed to list workspace files: ${response.statusText}`
        )
      }

      const rawData = await response.json()
      
      // Handle wrapped response format from agent
      let data = rawData
      if (rawData.body) {
        // Use the body field (now properly parsed JSON)
        data = rawData.body
      }
      
      return {
        files: data.files || [],
        path: data.path || path
      }
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Network error: ${error.message}`)
    }
  }

  /**
   * View file content
   */
  async viewFile(
    namespace: string,
    agentName: string,
    path: string
  ): Promise<string> {
    const url = this.buildWorkspaceUrl(namespace, agentName, '/files/view', { path })

    try {
      const response = await fetch(url)
      
      if (!response.ok) {
        throw new WorkspaceError(
          this.mapHttpStatusToErrorCode(response.status),
          `Failed to view file: ${response.statusText}`
        )
      }

      const rawData = await response.json()
      
      // Handle wrapped response format from agent
      let data = rawData
      if (rawData.body) {
        // Use the body field (now properly parsed JSON)
        data = rawData.body
      }
      
      return data.contents || data.content || ''
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Network error: ${error.message}`)
    }
  }

  /**
   * Upload file to workspace
   */
  async uploadFile(
    namespace: string,
    agentName: string,
    path: string,
    file: File | Buffer,
    filename?: string
  ): Promise<void> {
    const url = this.buildWorkspaceUrl(namespace, agentName, '/files')

    try {
      const formData = new FormData()
      
      if (file instanceof Buffer) {
        const blob = new Blob([new Uint8Array(file)])
        formData.append('file', blob, filename || 'upload.txt')
      } else {
        // file is File type
        formData.append('file', file as File)
      }
      
      formData.append('path', path)

      const response = await fetch(url, {
        method: 'POST',
        body: formData
      })
      
      if (!response.ok) {
        throw new WorkspaceError(
          this.mapHttpStatusToErrorCode(response.status),
          `Failed to upload file: ${response.statusText}`
        )
      }
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Network error: ${error.message}`)
    }
  }

  /**
   * Delete file or directory
   */
  async deleteFile(
    namespace: string,
    agentName: string,
    path: string
  ): Promise<void> {
    const url = this.buildWorkspaceUrl(namespace, agentName, '/files', { path })

    try {
      const response = await fetch(url, { method: 'DELETE' })
      
      if (!response.ok) {
        throw new WorkspaceError(
          this.mapHttpStatusToErrorCode(response.status),
          `Failed to delete file: ${response.statusText}`
        )
      }
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Network error: ${error.message}`)
    }
  }

  /**
   * Download file or directory as archive
   */
  async downloadFile(
    namespace: string,
    agentName: string,
    path: string
  ): Promise<Blob> {
    const url = this.buildWorkspaceUrl(namespace, agentName, '/files/download', { path })

    try {
      const response = await fetch(url)
      
      if (!response.ok) {
        throw new WorkspaceError(
          this.mapHttpStatusToErrorCode(response.status),
          `Failed to download file: ${response.statusText}`
        )
      }

      return await response.blob()
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Network error: ${error.message}`)
    }
  }

  /**
   * Build workspace API URL using service resolver
   */
  private buildWorkspaceUrl(
    namespace: string,
    agentName: string,
    endpoint: string,
    params?: Record<string, string>
  ): string {
    const baseUrl = this.serviceResolver.resolveServiceUrl({
      serviceName: agentName,
      namespace,
      port: 8080,
      path: `/api/v1/workspace${endpoint}`
    })

    if (params) {
      const searchParams = new URLSearchParams(params)
      return `${baseUrl}?${searchParams.toString()}`
    }

    return baseUrl
  }

  /**
   * Map HTTP status codes to workspace error codes
   */
  private mapHttpStatusToErrorCode(status: number): WorkspaceError['code'] {
    switch (status) {
      case 404:
        return 'NOT_FOUND'
      case 403:
        return 'ACCESS_DENIED'
      case 413:
        return 'SIZE_LIMIT'
      case 400:
        return 'INVALID_PATH'
      case 408:
        return 'TIMEOUT'
      default:
        return 'POD_ERROR'
    }
  }
}

// Export singleton instance
export const workspaceClient = new WorkspaceClient()