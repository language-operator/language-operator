import * as k8s from '@kubernetes/client-node'
import { k8sClient } from './k8s-client'

export interface WatchEvent<T = any> {
  type: 'ADDED' | 'MODIFIED' | 'DELETED' | 'ERROR'
  object?: T
  error?: string
  resourceVersion?: string
}

export interface WatchOptions {
  namespace: string
  labelSelector?: string
  fieldSelector?: string
  resourceVersion?: string
  timeoutSeconds?: number
}

export class KubernetesWatchService {
  private static instance: KubernetesWatchService
  private kc: k8s.KubeConfig
  private watchers = new Map<string, k8s.Watch>()
  private activeStreams = new Set<string>()

  private constructor() {
    this.kc = new k8s.KubeConfig()
    
    try {
      // Use same config logic as k8sClient
      if (process.env.KUBERNETES_SERVER_URL && process.env.KUBERNETES_TOKEN) {
        // Environment variable configuration with token
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
        // Docker Compose mode: use kubectl proxy (no token needed)
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
      } else if (process.env.NODE_ENV === 'development') {
        this.kc.loadFromDefault()
      } else {
        this.kc.loadFromCluster()
      }
    } catch (error) {
      console.error('❌ Failed to configure Kubernetes watch client:', error)
      throw new Error('Kubernetes configuration is required for watch service')
    }
  }

  public static getInstance(): KubernetesWatchService {
    if (!KubernetesWatchService.instance) {
      KubernetesWatchService.instance = new KubernetesWatchService()
    }
    return KubernetesWatchService.instance
  }

  /**
   * Watch custom resources (LanguageAgent, LanguageCluster, etc.)
   */
  async watchCustomResource<T = any>(
    group: string,
    version: string,
    plural: string,
    options: WatchOptions,
    onEvent: (event: WatchEvent<T>) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    const watchKey = `${group}/${version}/${plural}/${options.namespace}`
    
    // Close existing watcher if any
    this.stopWatch(watchKey)

    const watch = new k8s.Watch(this.kc)
    this.watchers.set(watchKey, watch)
    this.activeStreams.add(watchKey)

    const path = `/apis/${group}/${version}/namespaces/${options.namespace}/${plural}`
    const queryParams = new URLSearchParams()
    
    if (options.labelSelector) queryParams.append('labelSelector', options.labelSelector)
    if (options.fieldSelector) queryParams.append('fieldSelector', options.fieldSelector)
    if (options.resourceVersion) queryParams.append('resourceVersion', options.resourceVersion)
    if (options.timeoutSeconds) queryParams.append('timeoutSeconds', options.timeoutSeconds.toString())
    queryParams.append('watch', 'true')

    const fullPath = `${path}?${queryParams.toString()}`

    try {
      console.log(`🔍 Starting watch for ${plural} in namespace ${options.namespace}`)
      
      const stream = await watch.watch(
        fullPath,
        {},
        (type: string, apiObj: any, watchObj: any) => {
          if (!this.activeStreams.has(watchKey)) {
            // Stream was cancelled, ignore events
            return
          }

          const event: WatchEvent<T> = {
            type: type as WatchEvent<T>['type'],
            object: apiObj,
            resourceVersion: apiObj?.metadata?.resourceVersion
          }

          // Handle ERROR type specifically
          if (type === 'ERROR') {
            const errorMessage = apiObj?.message || apiObj?.reason || 'Unknown watch error'
            event.error = errorMessage
            console.error(`Watch error for ${plural}:`, errorMessage)
            
            if (onError) {
              onError(new Error(errorMessage))
            }
          }

          onEvent(event)
        },
        (error: any) => {
          console.error(`Watch stream ended for ${plural}:`, error)
          this.activeStreams.delete(watchKey)

          if (onError) {
            onError(error || new Error('Watch stream ended'))
          }

          // Note: Reconnection is now handled by the route layer
          // This prevents double reconnection logic and reconnection storms
        }
      )

      // Return cleanup function
      return () => this.stopWatch(watchKey)

    } catch (error) {
      console.error(`Failed to start watch for ${plural}:`, error)
      this.activeStreams.delete(watchKey)
      this.watchers.delete(watchKey)
      
      if (onError) {
        onError(error instanceof Error ? error : new Error(String(error)))
      }
      
      // Return no-op cleanup function
      return () => {}
    }
  }

  /**
   * Watch Language Agents
   */
  async watchLanguageAgents(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCustomResource(
      'langop.io',
      'v1alpha1',
      'languageagents',
      options,
      onEvent,
      onError
    )
  }

  /**
   * Watch Language Clusters
   */
  async watchLanguageClusters(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCustomResource(
      'langop.io',
      'v1alpha1',
      'languageclusters',
      options,
      onEvent,
      onError
    )
  }

  /**
   * Watch Language Models
   */
  async watchLanguageModels(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCustomResource(
      'langop.io',
      'v1alpha1',
      'languagemodels',
      options,
      onEvent,
      onError
    )
  }

  /**
   * Watch Language Tools
   */
  async watchLanguageTools(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCustomResource(
      'langop.io',
      'v1alpha1',
      'languagetools',
      options,
      onEvent,
      onError
    )
  }

  /**
   * Watch Language Personas
   */
  async watchLanguagePersonas(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCustomResource(
      'langop.io',
      'v1alpha1',
      'languagepersonas',
      options,
      onEvent,
      onError
    )
  }

  /**
   * Watch core Kubernetes resources (Pods, Events, etc.)
   */
  async watchCoreResource<T = any>(
    apiVersion: string,
    resource: string,
    options: WatchOptions,
    onEvent: (event: WatchEvent<T>) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    const watchKey = `core/${apiVersion}/${resource}/${options.namespace}`
    
    this.stopWatch(watchKey)

    const watch = new k8s.Watch(this.kc)
    this.watchers.set(watchKey, watch)
    this.activeStreams.add(watchKey)

    const path = `/api/${apiVersion}/namespaces/${options.namespace}/${resource}`
    const queryParams = new URLSearchParams()
    
    if (options.labelSelector) queryParams.append('labelSelector', options.labelSelector)
    if (options.fieldSelector) queryParams.append('fieldSelector', options.fieldSelector)
    if (options.resourceVersion) queryParams.append('resourceVersion', options.resourceVersion)
    if (options.timeoutSeconds) queryParams.append('timeoutSeconds', options.timeoutSeconds.toString())
    queryParams.append('watch', 'true')

    const fullPath = `${path}?${queryParams.toString()}`

    try {
      console.log(`🔍 Starting watch for ${resource} in namespace ${options.namespace}`)
      
      await watch.watch(
        fullPath,
        {},
        (type: string, apiObj: any) => {
          if (!this.activeStreams.has(watchKey)) {
            return
          }

          const event: WatchEvent<T> = {
            type: type as WatchEvent<T>['type'],
            object: apiObj,
            resourceVersion: apiObj?.metadata?.resourceVersion
          }

          if (type === 'ERROR') {
            const errorMessage = apiObj?.message || apiObj?.reason || 'Unknown watch error'
            event.error = errorMessage
            console.error(`Watch error for ${resource}:`, errorMessage)
            
            if (onError) {
              onError(new Error(errorMessage))
            }
          }

          onEvent(event)
        },
        (error: any) => {
          console.error(`Watch stream ended for ${resource}:`, error)
          this.activeStreams.delete(watchKey)

          if (onError) {
            onError(error || new Error('Watch stream ended'))
          }

          // Note: Reconnection is now handled by the route layer
          // This prevents double reconnection logic and reconnection storms
        }
      )

      return () => this.stopWatch(watchKey)

    } catch (error) {
      console.error(`Failed to start watch for ${resource}:`, error)
      this.activeStreams.delete(watchKey)
      this.watchers.delete(watchKey)
      
      if (onError) {
        onError(error instanceof Error ? error : new Error(String(error)))
      }
      
      return () => {}
    }
  }

  /**
   * Watch Pods (useful for agent logs and status)
   */
  async watchPods(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCoreResource('v1', 'pods', options, onEvent, onError)
  }

  /**
   * Watch Events (for debugging and monitoring)
   */
  async watchEvents(
    options: WatchOptions,
    onEvent: (event: WatchEvent) => void,
    onError?: (error: Error) => void
  ): Promise<() => void> {
    return this.watchCoreResource('v1', 'events', options, onEvent, onError)
  }

  /**
   * Stop a specific watch
   */
  private stopWatch(watchKey: string): void {
    this.activeStreams.delete(watchKey)
    const watcher = this.watchers.get(watchKey)
    if (watcher) {
      try {
        // The k8s watch client doesn't have a direct close method,
        // but removing from activeStreams will stop processing events
        this.watchers.delete(watchKey)
        console.log(`🛑 Stopped watch: ${watchKey}`)
      } catch (error) {
        console.error(`Error stopping watch ${watchKey}:`, error)
      }
    }
  }

  /**
   * Stop all active watches (cleanup)
   */
  public stopAllWatches(): void {
    console.log('🛑 Stopping all watches...')
    this.activeStreams.clear()
    for (const [key] of this.watchers) {
      this.stopWatch(key)
    }
  }

  /**
   * Get active watch count (for debugging)
   */
  public getActiveWatchCount(): number {
    return this.activeStreams.size
  }

  /**
   * List active watches (for debugging)
   */
  public getActiveWatches(): string[] {
    return Array.from(this.activeStreams)
  }
}

// Export singleton instance
export const watchService = KubernetesWatchService.getInstance()