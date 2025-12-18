import { k8sClient } from './k8s-client'
import { 
  ClusterNotFoundError, 
  ClusterAccessDeniedError, 
  InvalidClusterNameError,
  OrphanedResourceError,
  validateClusterNameFormat,
  handleKubernetesOperation
} from './api-error-handler'

export interface ClusterValidationOptions {
  requireClusterRef?: boolean
  allowOrphanedResources?: boolean
  validateAccess?: boolean
}

export interface ClusterValidationResult {
  exists: boolean
  accessible: boolean
  cluster?: any
}

export async function validateClusterExists(
  namespace: string,
  clusterName: string,
  options: ClusterValidationOptions = {}
): Promise<ClusterValidationResult> {
  // First validate the cluster name format
  validateClusterNameFormat(clusterName)

  try {
    // Check if cluster exists in the namespace
    const cluster = await handleKubernetesOperation(
      `get cluster '${clusterName}'`,
      k8sClient.getLanguageCluster(namespace, clusterName)
    )

    if (!cluster) {
      throw new ClusterNotFoundError(clusterName, namespace)
    }

    // Validate cluster is accessible (not in failed state, etc.)
    const accessible = validateClusterAccessibility(cluster)
    
    if (options.validateAccess && !accessible) {
      throw new ClusterAccessDeniedError(
        clusterName, 
        namespace,
        `Cluster is in '${cluster.status?.phase || 'unknown'}' state`
      )
    }

    return {
      exists: true,
      accessible,
      cluster
    }

  } catch (error) {
    // Re-throw our custom errors
    if (error instanceof ClusterNotFoundError || error instanceof ClusterAccessDeniedError) {
      throw error
    }

    // Handle Kubernetes errors
    console.error(`Error validating cluster ${clusterName}:`, error)
    throw new ClusterNotFoundError(clusterName, namespace, 'Failed to validate cluster existence')
  }
}

export function validateClusterAccessibility(cluster: any): boolean {
  if (!cluster?.status) {
    return true // Allow clusters without status (newly created)
  }

  const phase = cluster.status.phase
  
  // Allow access to clusters in these phases
  const accessiblePhases = ['Ready', 'Pending', 'Scaling']
  
  return accessiblePhases.includes(phase)
}

export async function validateClusterAccess(
  namespace: string,
  clusterName: string,
  organizationId: string,
  userRole?: string
): Promise<void> {
  const result = await validateClusterExists(namespace, clusterName, { validateAccess: true })
  
  if (!result.cluster) {
    throw new ClusterNotFoundError(clusterName, namespace)
  }

  // Check organization ownership via labels
  const clusterOrgId = result.cluster.metadata?.labels?.['langop.io/organization-id']
  
  if (clusterOrgId !== organizationId) {
    throw new ClusterAccessDeniedError(
      clusterName, 
      namespace,
      `Cluster belongs to different organization`
    )
  }
}

export function validateClusterRef(
  resource: any,
  expectedClusterName: string,
  options: ClusterValidationOptions = {}
): void {
  if (!resource) {
    throw new Error('Resource is required for cluster reference validation')
  }

  const resourceClusterRef = resource.spec?.clusterRef
  const resourceName = resource.metadata?.name || 'unknown'
  const resourceType = resource.kind || 'Resource'

  // If resource has clusterRef, it must match expected cluster
  if (resourceClusterRef && resourceClusterRef !== expectedClusterName) {
    // Always throw an error for mismatched clusterRef - this resource belongs to a different cluster
    throw new OrphanedResourceError(resourceType, resourceName, resourceClusterRef)
  }
  
  // Check if clusterRef is required
  if (options.requireClusterRef && !resourceClusterRef) {
    throw new InvalidClusterNameError(
      '',
      `${resourceType} '${resourceName}' is missing required clusterRef field`
    )
  }
  
  // If resource has no clusterRef, handle based on requirements
  if (!resourceClusterRef) {
    if (options.requireClusterRef) {
      // Already handled above
      return
    }
    if (!options.allowOrphanedResources) {
      throw new OrphanedResourceError(resourceType, resourceName, 'none')
    }
    // If allowOrphanedResources is true, we still exclude resources without clusterRef
    // from cluster-scoped views to maintain proper separation
    console.warn(
      `Excluding ${resourceType}/${resourceName} with no clusterRef from cluster '${expectedClusterName}'`
    )
    throw new OrphanedResourceError(resourceType, resourceName, 'none')
  }
}

export function validateResourceBelongsToCluster(
  resources: any[],
  clusterName: string,
  options: ClusterValidationOptions = {}
): any[] {
  return resources.filter(resource => {
    try {
      validateClusterRef(resource, clusterName, options)
      return true
    } catch (error) {
      if (error instanceof OrphanedResourceError && options.allowOrphanedResources) {
        return false // Exclude orphaned resources
      }
      throw error // Re-throw other validation errors
    }
  })
}

export async function validateClusterForResourceCreation(
  namespace: string,
  clusterName: string,
  organizationId: string,
  resourceType: string
): Promise<void> {
  // Validate cluster exists and is accessible
  const result = await validateClusterExists(namespace, clusterName, { validateAccess: true })
  
  if (!result.accessible) {
    throw new ClusterAccessDeniedError(
      clusterName,
      namespace, 
      `Cannot create ${resourceType} in inaccessible cluster`
    )
  }

  // Validate organization access
  await validateClusterAccess(namespace, clusterName, organizationId)
  
  // Check if cluster is in a state that allows resource creation
  const cluster = result.cluster
  const phase = cluster?.status?.phase
  
  if (phase === 'Failed') {
    throw new ClusterAccessDeniedError(
      clusterName,
      namespace,
      `Cannot create ${resourceType} in failed cluster`
    )
  }
}

export function sanitizeClusterName(clusterName: string): string {
  if (!clusterName) return ''
  
  // Remove any characters that aren't lowercase alphanumeric or hyphens
  return clusterName
    .toLowerCase()
    .replace(/[^a-z0-9-]/g, '')
    .replace(/^-+|-+$/g, '') // Remove leading/trailing hyphens
    .slice(0, 253) // Enforce max length
}

export function isValidClusterName(clusterName: string): boolean {
  try {
    validateClusterNameFormat(clusterName)
    return true
  } catch {
    return false
  }
}

export async function getClusterResourceCounts(
  namespace: string,
  clusterName: string,
  organizationId: string
): Promise<{
  agents: number
  models: number
  tools: number
  personas: number
}> {
  // Validate cluster access first
  await validateClusterAccess(namespace, clusterName, organizationId)

  const counts = {
    agents: 0,
    models: 0,
    tools: 0,
    personas: 0
  }

  try {
    // Count agents for this cluster
    const agentsResponse = await handleKubernetesOperation(
      'list agents',
      k8sClient.listLanguageAgents(namespace)
    )
    
    let agents = []
    if (agentsResponse.body && typeof agentsResponse.body === 'object') {
      agents = (agentsResponse.body as any)?.items || []
    } else if (agentsResponse.data && typeof agentsResponse.data === 'object') {
      agents = (agentsResponse.data as any)?.items || []
    } else if (Array.isArray(agentsResponse)) {
      agents = agentsResponse
    } else if ((agentsResponse as any)?.items) {
      agents = (agentsResponse as any).items
    }

    counts.agents = validateResourceBelongsToCluster(agents, clusterName, { 
      allowOrphanedResources: true 
    }).length

  } catch (error) {
    console.error(`Error counting agents for cluster ${clusterName}:`, error)
  }

  try {
    // Count models for this cluster
    const modelsResponse = await handleKubernetesOperation(
      'list models',
      k8sClient.listLanguageModels(namespace)
    )
    
    let models = []
    if (modelsResponse.body && typeof modelsResponse.body === 'object') {
      models = (modelsResponse.body as any)?.items || []
    } else if (modelsResponse.data && typeof modelsResponse.data === 'object') {
      models = (modelsResponse.data as any)?.items || []
    } else if (Array.isArray(modelsResponse)) {
      models = modelsResponse
    } else if ((modelsResponse as any)?.items) {
      models = (modelsResponse as any).items
    }

    counts.models = validateResourceBelongsToCluster(models, clusterName, { 
      allowOrphanedResources: true 
    }).length

  } catch (error) {
    console.error(`Error counting models for cluster ${clusterName}:`, error)
  }

  try {
    // Count tools for this cluster
    const toolsResponse = await handleKubernetesOperation(
      'list tools',
      k8sClient.listLanguageTools(namespace)
    )
    
    let tools = []
    if (toolsResponse.body && typeof toolsResponse.body === 'object') {
      tools = (toolsResponse.body as any)?.items || []
    } else if (toolsResponse.data && typeof toolsResponse.data === 'object') {
      tools = (toolsResponse.data as any)?.items || []
    } else if (Array.isArray(toolsResponse)) {
      tools = toolsResponse
    } else if ((toolsResponse as any)?.items) {
      tools = (toolsResponse as any).items
    }

    counts.tools = validateResourceBelongsToCluster(tools, clusterName, { 
      allowOrphanedResources: true 
    }).length

  } catch (error) {
    console.error(`Error counting tools for cluster ${clusterName}:`, error)
  }

  try {
    // Count personas for this cluster
    const personasResponse = await handleKubernetesOperation(
      'list personas',
      k8sClient.listLanguagePersonas(namespace)
    )
    
    let personas = []
    if (personasResponse.body && typeof personasResponse.body === 'object') {
      personas = (personasResponse.body as any)?.items || []
    } else if (personasResponse.data && typeof personasResponse.data === 'object') {
      personas = (personasResponse.data as any)?.items || []
    } else if (Array.isArray(personasResponse)) {
      personas = personasResponse
    } else if ((personasResponse as any)?.items) {
      personas = (personasResponse as any).items
    }

    counts.personas = validateResourceBelongsToCluster(personas, clusterName, { 
      allowOrphanedResources: true 
    }).length

  } catch (error) {
    console.error(`Error counting personas for cluster ${clusterName}:`, error)
  }

  return counts
}

