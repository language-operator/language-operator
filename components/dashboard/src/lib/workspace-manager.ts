import * as k8s from '@kubernetes/client-node'
import { 
  FileEntry, 
  WorkspacePodInfo, 
  WorkspaceError, 
  WORKSPACE_LIMITS, 
  HIDDEN_FILE_PATTERNS,
  VIEWABLE_FILE_EXTENSIONS 
} from '@/types/workspace'

class WorkspaceManager {
  private static instance: WorkspaceManager
  private kc: k8s.KubeConfig
  private coreV1Api: k8s.CoreV1Api
  private exec!: k8s.Exec

  private constructor() {
    this.kc = new k8s.KubeConfig()

    try {
      // Load config based on environment (same pattern as k8s-client.ts)
      if (process.env.KUBERNETES_SERVER_URL && process.env.KUBERNETES_TOKEN) {
        const cluster = {
          name: 'env-cluster',
          server: process.env.KUBERNETES_SERVER_URL,
          skipTLSVerify: process.env.KUBERNETES_SKIP_TLS_VERIFY === 'true',
        }
        const user = {
          name: 'env-user',
          token: process.env.KUBERNETES_TOKEN,
        }
        const context = {
          name: 'env-context',
          user: user.name,
          cluster: cluster.name,
        }
        this.kc.loadFromOptions({
          clusters: [cluster],
          users: [user],
          contexts: [context],
          currentContext: context.name,
        })
      } else if (process.env.KUBERNETES_SERVER_URL && process.env.KUBERNETES_SERVER_URL.includes('kubectl-proxy')) {
        // Docker Compose mode: use kubectl proxy for API calls but direct connection for exec
        const cluster = {
          name: 'kubectl-proxy-cluster',
          server: process.env.KUBERNETES_SERVER_URL,
          skipTLSVerify: true, // kubectl proxy doesn't use TLS
        }
        const user = {
          name: 'kubectl-proxy-user',
          // No token needed for kubectl proxy
        }
        const context = {
          name: 'kubectl-proxy-context',
          user: user.name,
          cluster: cluster.name,
        }
        this.kc.loadFromOptions({
          clusters: [cluster],
          users: [user],
          contexts: [context],
          currentContext: context.name,
        })

        // For exec operations, we need direct access since kubectl proxy doesn't support exec properly
        // Create a separate kubeconfig for exec operations
        const execKc = new k8s.KubeConfig()
        execKc.loadFromDefault() // Use direct connection for exec
        this.exec = new k8s.Exec(execKc)
      } else if (process.env.NODE_ENV === 'development') {
        this.kc.loadFromDefault()
      } else {
        this.kc.loadFromCluster()
      }

      this.coreV1Api = this.kc.makeApiClient(k8s.CoreV1Api)
      
      // Only create exec if it wasn't created above in kubectl-proxy mode
      if (!this.exec) {
        this.exec = new k8s.Exec(this.kc)
      }
    } catch (error) {
      console.error('❌ Failed to configure WorkspaceManager:', error instanceof Error ? error.message : String(error))
      throw new Error('Kubernetes configuration is required for workspace operations')
    }
  }

  public static getInstance(): WorkspaceManager {
    if (!WorkspaceManager.instance) {
      WorkspaceManager.instance = new WorkspaceManager()
    }
    return WorkspaceManager.instance
  }

  /**
   * Delete a workspace pod
   */
  private async deleteWorkspacePod(namespace: string, podName: string): Promise<void> {
    try {
      await this.coreV1Api.deleteNamespacedPod({
        name: podName,
        namespace,
      })
      console.log(`Workspace Manager - Deleted expired pod: ${podName}`)
    } catch (error: any) {
      // Check for 404 - pod already deleted
      const is404 = error.response?.statusCode === 404 || 
                   error.statusCode === 404 ||
                   error.response?.status === 404 ||
                   error.status === 404 ||
                   error.code === 404 ||
                   (error.response?.body && error.response.body.code === 404) ||
                   (error.body && error.body.code === 404)
      
      if (is404) {
        console.log(`Workspace Manager - Pod ${podName} already deleted`)
        return
      }
      
      console.error(`Workspace Manager - Failed to delete pod ${podName}:`, error.message)
      throw new WorkspaceError('POD_ERROR', `Failed to delete expired pod: ${error.message}`)
    }
  }

  /**
   * Ensure a file manager pod exists for the given agent
   */
  async ensureFileManagerPod(namespace: string, agentName: string): Promise<WorkspacePodInfo> {
    const podName = `${agentName}-workspace-manager`
    
    try {
      // Check if pod already exists
      const existingPod = await this.coreV1Api.readNamespacedPod({
        name: podName,
        namespace,
      })

      const pod = (existingPod as any).body || existingPod
      const phase = pod.status?.phase
      const createdAt = pod.metadata?.creationTimestamp || new Date().toISOString()
      const age = Date.now() - Date.parse(createdAt)
      const maxAge = WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60 * 1000

      // Delete pod if it's expired, failed, or succeeded
      if (phase === 'Succeeded' || phase === 'Failed' || age > maxAge) {
        console.log(`Workspace Manager - Pod ${podName} is expired/failed (phase: ${phase}, age: ${Math.round(age / 60000)}min), deleting...`)
        await this.deleteWorkspacePod(namespace, podName)
        return await this.createFileManagerPod(namespace, agentName)
      }

      // Pod exists and is still valid
      const expiresAt = new Date(Date.parse(createdAt) + WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60 * 1000).toISOString()
      console.log(`Workspace Manager - Using existing pod ${podName} (phase: ${phase})`)
      
      return {
        name: podName,
        namespace,
        status: this.mapPodPhase(phase),
        createdAt,
        expiresAt,
      }
    } catch (error: any) {
      // Check for 404 in multiple possible locations based on k8s client error structure
      const is404 = error.response?.statusCode === 404 || 
                   error.statusCode === 404 ||
                   error.response?.status === 404 ||
                   error.status === 404 ||
                   error.code === 404 ||
                   (error.response?.body && error.response.body.code === 404) ||
                   (error.body && error.body.code === 404)
      
      if (!is404) {
        console.error('Workspace Manager - Pod check error:', {
          message: error.message,
          statusCode: error.statusCode,
          responseStatusCode: error.response?.statusCode,
          responseStatus: error.response?.status,
          errorStructure: JSON.stringify(error, null, 2)
        })
        throw new WorkspaceError('POD_ERROR', `Failed to check pod status: ${error.message}`)
      }
      console.log('Workspace Manager - Pod not found (404), will create it')
      // Pod doesn't exist (404), continue to create it
    }

    // Pod doesn't exist, create it
    return await this.createFileManagerPod(namespace, agentName)
  }

  private async createFileManagerPod(namespace: string, agentName: string): Promise<WorkspacePodInfo> {
    const podName = `${agentName}-workspace-manager`
    const workspacePvcName = `${agentName}-workspace`

    const podSpec: k8s.V1Pod = {
      metadata: {
        name: podName,
        namespace,
        labels: {
          'app.kubernetes.io/name': 'workspace-manager',
          'app.kubernetes.io/component': 'file-manager',
          'app.kubernetes.io/instance': agentName,
          'langop.io/agent': agentName,
          'langop.io/managed-by': 'dashboard',
        },
        annotations: {
          'langop.io/created-at': new Date().toISOString(),
          'langop.io/expires-at': new Date(Date.now() + WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60 * 1000).toISOString(),
        },
      },
      spec: {
        restartPolicy: 'Never',
        securityContext: {
          runAsNonRoot: true,
          runAsUser: 1000,
          runAsGroup: 1000,
          fsGroup: 1000,
        },
        containers: [
          {
            name: 'file-manager',
            image: 'alpine:3.18',
            command: ['/bin/sh', '-c'],
            args: [`
              # Install required tools
              apk add --no-cache findutils coreutils
              # Keep container running for file operations
              sleep ${WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60}
            `],
            volumeMounts: [
              {
                name: 'workspace',
                mountPath: '/workspace',
              },
              {
                name: 'tmp',
                mountPath: '/tmp',
              },
            ],
            resources: {
              limits: {
                cpu: '50m',
                memory: '64Mi',
              },
              requests: {
                cpu: '10m',
                memory: '32Mi',
              },
            },
            securityContext: {
              readOnlyRootFilesystem: true,
              allowPrivilegeEscalation: false,
              capabilities: {
                drop: ['ALL'],
              },
            },
          },
        ],
        volumes: [
          {
            name: 'workspace',
            persistentVolumeClaim: {
              claimName: workspacePvcName,
            },
          },
          {
            name: 'tmp',
            emptyDir: {
              medium: 'Memory',
            },
          },
        ],
        // Auto-delete the pod after timeout
        activeDeadlineSeconds: WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60,
      },
    }

    try {
      const response = await this.coreV1Api.createNamespacedPod({
        namespace,
        body: podSpec,
      })

      const pod = (response as any).body || response
      const createdAt = pod.metadata?.creationTimestamp || new Date().toISOString()
      const expiresAt = new Date(Date.parse(createdAt) + WORKSPACE_LIMITS.POD_TIMEOUT_MINUTES * 60 * 1000).toISOString()

      return {
        name: podName,
        namespace,
        status: this.mapPodPhase(pod.status?.phase),
        createdAt,
        expiresAt,
      }
    } catch (error: any) {
      throw new WorkspaceError('POD_ERROR', `Failed to create file manager pod: ${error.message}`)
    }
  }

  /**
   * Execute a command in the file manager pod
   */
  async executeCommand(
    namespace: string, 
    podName: string, 
    command: string[], 
    options?: { timeout?: number }
  ): Promise<string> {
    const timeoutMs = options?.timeout || 30000 // 30 second default

    try {
      return await Promise.race([
        this.execInPod(namespace, podName, command),
        new Promise<never>((_, reject) => 
          setTimeout(() => reject(new WorkspaceError('TIMEOUT', 'Command execution timeout')), timeoutMs)
        )
      ])
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Command execution failed: ${error.message}`)
    }
  }

  private async execInPod(namespace: string, podName: string, command: string[]): Promise<string> {
    const { Writable } = require('stream')
    
    return new Promise((resolve, reject) => {
      let output = ''
      let errorOutput = ''

      // Create writable streams to capture output
      const stdout = new Writable({
        write(chunk: any, encoding: any, callback: any) {
          output += chunk.toString()
          callback()
        }
      })

      const stderr = new Writable({
        write(chunk: any, encoding: any, callback: any) {
          errorOutput += chunk.toString()
          callback()
        }
      })

      this.exec.exec(
        namespace,
        podName,
        'file-manager',
        command,
        stdout,
        stderr,
        process.stdin,
        true,
        (status: any) => {
          if (status.status === 'Success') {
            resolve(output.trim())
          } else {
            reject(new Error(`Command failed (${status.status}): ${errorOutput || status.message}`))
          }
        }
      ).catch((error) => {
        console.error('Workspace Manager - Exec error:', error)
        reject(error)
      })
    })
  }

  /**
   * List files in a directory
   */
  async listDirectory(namespace: string, agentName: string, path: string = '/'): Promise<FileEntry[]> {
    this.validatePath(path)
    
    const podInfo = await this.ensureFileManagerPod(namespace, agentName)
    await this.waitForPodReady(namespace, podInfo.name, 30000)

    const safePath = this.sanitizePath(path)
    const fullPath = `/workspace${safePath}`

    try {
      // Use ls -la to get detailed file information
      const output = await this.executeCommand(
        namespace, 
        podInfo.name, 
        ['sh', '-c', `ls -la "${fullPath}" 2>/dev/null || echo "ERROR: Directory not accessible"`]
      )

      if (output.includes('ERROR: Directory not accessible')) {
        throw new WorkspaceError('NOT_FOUND', `Directory not found: ${path}`)
      }

      return this.parseDirectoryListing(output, path)
    } catch (error: any) {
      if (error instanceof WorkspaceError) {
        throw error
      }
      throw new WorkspaceError('POD_ERROR', `Failed to list directory: ${error.message}`)
    }
  }

  /**
   * Get file content for viewing
   */
  async viewFile(namespace: string, agentName: string, path: string): Promise<string> {
    this.validatePath(path)
    
    const podInfo = await this.ensureFileManagerPod(namespace, agentName)
    await this.waitForPodReady(namespace, podInfo.name, 30000)

    const safePath = this.sanitizePath(path)
    const fullPath = `/workspace${safePath}`

    // Check file size first
    const sizeOutput = await this.executeCommand(
      namespace,
      podInfo.name,
      ['sh', '-c', `stat -c %s "${fullPath}" 2>/dev/null || echo "ERROR"`]
    )

    if (sizeOutput.includes('ERROR')) {
      throw new WorkspaceError('NOT_FOUND', `File not found: ${path}`)
    }

    const size = parseInt(sizeOutput.trim())
    if (size > WORKSPACE_LIMITS.MAX_VIEW_SIZE) {
      throw new WorkspaceError('SIZE_LIMIT', `File too large to view: ${size} bytes`)
    }

    // Read file content
    try {
      const content = await this.executeCommand(
        namespace,
        podInfo.name,
        ['cat', fullPath]
      )
      return content
    } catch (error: any) {
      throw new WorkspaceError('POD_ERROR', `Failed to read file: ${error.message}`)
    }
  }

  /**
   * Download file as base64
   */
  async downloadFile(namespace: string, agentName: string, path: string): Promise<Buffer> {
    this.validatePath(path)
    
    const podInfo = await this.ensureFileManagerPod(namespace, agentName)
    await this.waitForPodReady(namespace, podInfo.name, 30000)

    const safePath = this.sanitizePath(path)
    const fullPath = `/workspace${safePath}`

    try {
      const base64Output = await this.executeCommand(
        namespace,
        podInfo.name,
        ['sh', '-c', `base64 -w 0 "${fullPath}"`]
      )
      
      return Buffer.from(base64Output.trim(), 'base64')
    } catch (error: any) {
      throw new WorkspaceError('POD_ERROR', `Failed to download file: ${error.message}`)
    }
  }

  /**
   * Upload file from base64 content
   */
  async uploadFile(
    namespace: string, 
    agentName: string, 
    path: string, 
    content: Buffer
  ): Promise<void> {
    this.validatePath(path)
    
    if (content.length > WORKSPACE_LIMITS.MAX_UPLOAD_SIZE) {
      throw new WorkspaceError('SIZE_LIMIT', `File too large to upload: ${content.length} bytes`)
    }

    const podInfo = await this.ensureFileManagerPod(namespace, agentName)
    await this.waitForPodReady(namespace, podInfo.name, 30000)

    const safePath = this.sanitizePath(path)
    const fullPath = `/workspace${safePath}`
    const base64Content = content.toString('base64')

    try {
      // Create directory if needed
      const dirPath = fullPath.substring(0, fullPath.lastIndexOf('/'))
      if (dirPath !== '/workspace') {
        await this.executeCommand(
          namespace,
          podInfo.name,
          ['mkdir', '-p', dirPath]
        )
      }

      // Write file from base64
      await this.executeCommand(
        namespace,
        podInfo.name,
        ['sh', '-c', `echo '${base64Content}' | base64 -d > "${fullPath}"`]
      )
    } catch (error: any) {
      throw new WorkspaceError('POD_ERROR', `Failed to upload file: ${error.message}`)
    }
  }

  // Helper methods

  private validatePath(path: string): void {
    if (!path || path.includes('..') || path.includes('\0')) {
      throw new WorkspaceError('INVALID_PATH', `Invalid file path: ${path}`)
    }
  }

  private sanitizePath(path: string): string {
    // Ensure path starts with /
    let safePath = path.startsWith('/') ? path : `/${path}`
    // Remove any double slashes
    safePath = safePath.replace(/\/+/g, '/')
    // Remove trailing slash unless it's root
    if (safePath !== '/' && safePath.endsWith('/')) {
      safePath = safePath.slice(0, -1)
    }
    return safePath
  }

  private async waitForPodReady(namespace: string, podName: string, timeoutMs: number): Promise<void> {
    const startTime = Date.now()
    
    while (Date.now() - startTime < timeoutMs) {
      try {
        const pod = await this.coreV1Api.readNamespacedPod({ name: podName, namespace })
        const podData = (pod as any).body || pod
        const phase = podData.status?.phase
        
        if (phase === 'Running') {
          return
        }
        
        if (phase === 'Failed' || phase === 'Succeeded') {
          throw new WorkspaceError('POD_ERROR', `Pod failed to start: ${phase}`)
        }
        
        // Wait 1 second before checking again
        await new Promise(resolve => setTimeout(resolve, 1000))
      } catch (error: any) {
        throw new WorkspaceError('POD_ERROR', `Pod status check failed: ${error.message}`)
      }
    }
    
    throw new WorkspaceError('TIMEOUT', 'Timeout waiting for pod to be ready')
  }

  private parseDirectoryListing(output: string, basePath: string): FileEntry[] {
    const lines = output.trim().split('\n').filter(line => {
      // Skip total line and current/parent directory entries
      return line && !line.startsWith('total') && !line.endsWith(' .') && !line.endsWith(' ..')
    })

    return lines
      .map(line => this.parseFileEntry(line, basePath))
      .filter((entry): entry is FileEntry => entry !== null)
      .filter(entry => !this.isHiddenFile(entry.name))
      .filter(entry => entry.name !== '.' && entry.name !== '..') // Filter out . and .. entries
  }

  private parseFileEntry(line: string, basePath: string): FileEntry | null {
    // Strip ANSI color codes first
    const cleanLine = line.replace(/\x1b\[[0-9;]*m/g, '')
    
    // Parse ls -la output: permissions links owner group size month day time/year name
    const parts = cleanLine.trim().split(/\s+/)
    if (parts.length < 9) return null

    const permissions = parts[0]
    const size = parseInt(parts[4]) || 0
    const name = parts.slice(8).join(' ') // Handle filenames with spaces
    const type = permissions.startsWith('d') ? 'directory' : 'file'
    
    // Construct full path
    const fullPath = basePath === '/' ? `/${name}` : `${basePath}/${name}`

    return {
      name,
      path: fullPath,
      type,
      size: type === 'file' ? size : undefined,
      permissions: permissions.slice(1), // Remove type indicator
      modified: `${parts[5]} ${parts[6]} ${parts[7]}`, // Basic timestamp
    }
  }

  private isHiddenFile(filename: string): boolean {
    return HIDDEN_FILE_PATTERNS.some(pattern => pattern.test(filename))
  }

  private mapPodPhase(phase?: string): WorkspacePodInfo['status'] {
    switch (phase) {
      case 'Running': return 'running'
      case 'Succeeded': return 'succeeded'
      case 'Failed': return 'failed'
      default: return 'pending'
    }
  }


  /**
   * Clean up expired file manager pods
   */
  async cleanupExpiredPods(namespace: string): Promise<void> {
    try {
      const pods = await this.coreV1Api.listNamespacedPod({
        namespace,
        labelSelector: 'app.kubernetes.io/name=workspace-manager',
      })

      const podsData = (pods as any).body || pods
      const now = Date.now()
      const expiredPods = podsData.items.filter((pod: any) => {
        const expiresAt = pod.metadata?.annotations?.['langop.io/expires-at']
        return expiresAt && Date.parse(expiresAt) < now
      })

      for (const pod of expiredPods) {
        if (pod.metadata?.name) {
          try {
            await this.coreV1Api.deleteNamespacedPod({
              name: pod.metadata.name,
              namespace,
            })
            console.log(`Cleaned up expired workspace pod: ${pod.metadata.name}`)
          } catch (error: any) {
            console.error(`Failed to cleanup pod ${pod.metadata.name}:`, error.message)
          }
        }
      }
    } catch (error: any) {
      console.error('Failed to cleanup expired workspace pods:', error.message)
    }
  }
}

// Export singleton instance
export const workspaceManager = WorkspaceManager.getInstance()