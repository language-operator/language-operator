/**
 * Utility functions for cluster-scoped resource filtering
 */

/**
 * Standard cluster filtering logic for all resource types.
 * Filters resources to only include those that belong to the specified cluster.
 * 
 * @param resources Array of Kubernetes resources with spec.clusterRef property
 * @param clusterName Name of the cluster to filter by
 * @returns Filtered array containing only resources belonging to the cluster
 */
export function filterByClusterRef<T>(
  resources: T[],
  clusterName: string
): T[] {
  return resources.filter((resource: any) => resource.spec?.clusterRef === clusterName)
}

/**
 * Type guard to check if a resource has cluster reference
 */
export function hasClusterRef(resource: any): resource is { spec: { clusterRef: string } } {
  return resource?.spec?.clusterRef && typeof resource.spec.clusterRef === 'string'
}