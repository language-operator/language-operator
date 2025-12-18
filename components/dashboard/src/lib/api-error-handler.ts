import { NextResponse } from 'next/server'
import { z } from 'zod'

export type ApiErrorCode = 
  | 'CLUSTER_NOT_FOUND'
  | 'CLUSTER_ACCESS_DENIED'
  | 'INVALID_CLUSTER_NAME'
  | 'RESOURCE_NOT_FOUND'
  | 'VALIDATION_ERROR'
  | 'PERMISSION_DENIED'
  | 'KUBERNETES_ERROR'
  | 'AUTHENTICATION_REQUIRED'
  | 'RESOURCE_CONFLICT'
  | 'INVALID_INPUT'
  | 'INTERNAL_ERROR'
  | 'TIMEOUT_ERROR'
  | 'ORPHANED_RESOURCE'

export interface ApiErrorResponse {
  error: string
  code: ApiErrorCode
  details?: string
  context?: Record<string, any>
  timestamp?: string
}

export interface ApiSuccessResponse<T = any> {
  success: true
  data: T
  message?: string
  [key: string]: any
}

export class ApiError extends Error {
  public readonly code: ApiErrorCode
  public readonly statusCode: number
  public readonly details?: string
  public readonly context?: Record<string, any>

  constructor(
    message: string,
    code: ApiErrorCode,
    statusCode: number,
    details?: string,
    context?: Record<string, any>
  ) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.statusCode = statusCode
    this.details = details
    this.context = context
  }
}

export class ClusterNotFoundError extends ApiError {
  constructor(clusterName: string, namespace: string, details?: string) {
    super(
      `Cluster '${clusterName}' not found`,
      'CLUSTER_NOT_FOUND',
      404,
      details || `The cluster '${clusterName}' does not exist in namespace '${namespace}' or you don't have access to it`,
      { clusterName, namespace }
    )
  }
}

export class ClusterAccessDeniedError extends ApiError {
  constructor(clusterName: string, namespace: string, userRole?: string) {
    super(
      `Access denied to cluster '${clusterName}'`,
      'CLUSTER_ACCESS_DENIED', 
      403,
      `You don't have permission to access cluster '${clusterName}' in namespace '${namespace}'`,
      { clusterName, namespace, userRole }
    )
  }
}

export class InvalidClusterNameError extends ApiError {
  constructor(clusterName: string, details?: string) {
    super(
      `Invalid cluster name '${clusterName}'`,
      'INVALID_CLUSTER_NAME',
      400,
      details || `Cluster name '${clusterName}' contains invalid characters or format`,
      { clusterName }
    )
  }
}

export class ValidationError extends ApiError {
  constructor(details: string, zodError?: z.ZodError) {
    const context = zodError ? {
      validationErrors: zodError.issues.map(issue => ({
        field: issue.path.join('.'),
        message: issue.message,
        code: issue.code
      }))
    } : undefined

    super(
      'Validation failed',
      'VALIDATION_ERROR',
      400,
      details,
      context
    )
  }
}

export class ResourceNotFoundError extends ApiError {
  constructor(resourceType: string, resourceName: string, namespace?: string) {
    super(
      `${resourceType} '${resourceName}' not found`,
      'RESOURCE_NOT_FOUND',
      404,
      `The ${resourceType.toLowerCase()} '${resourceName}' does not exist${namespace ? ` in namespace '${namespace}'` : ''}`,
      { resourceType, resourceName, namespace }
    )
  }
}

export class KubernetesError extends ApiError {
  constructor(operation: string, error: any) {
    const isNotFound = error?.response?.statusCode === 404 || error?.statusCode === 404
    const isForbidden = error?.response?.statusCode === 403 || error?.statusCode === 403
    const isTimeout = error?.code === 'ETIMEDOUT' || error?.code === 'ECONNRESET'
    
    let message = `Kubernetes ${operation} failed`
    let code: ApiErrorCode = 'KUBERNETES_ERROR'
    let statusCode = 500
    
    if (isNotFound) {
      message = `Resource not found during ${operation}`
      code = 'RESOURCE_NOT_FOUND'
      statusCode = 404
    } else if (isForbidden) {
      message = `Permission denied for ${operation}`
      code = 'PERMISSION_DENIED'  
      statusCode = 403
    } else if (isTimeout) {
      message = `Timeout during ${operation}`
      code = 'TIMEOUT_ERROR'
      statusCode = 504
    }

    super(
      message,
      code,
      statusCode,
      error?.message || error?.response?.body?.message || 'Unknown Kubernetes error',
      { 
        operation,
        k8sStatusCode: error?.response?.statusCode || error?.statusCode,
        k8sReason: error?.response?.body?.reason,
        k8sCode: error?.code
      }
    )
  }
}

export class OrphanedResourceError extends ApiError {
  constructor(resourceType: string, resourceName: string, clusterName: string) {
    super(
      `${resourceType} '${resourceName}' is orphaned`,
      'ORPHANED_RESOURCE',
      409,
      `The ${resourceType.toLowerCase()} '${resourceName}' references cluster '${clusterName}' which no longer exists`,
      { resourceType, resourceName, clusterName }
    )
  }
}

export function createErrorResponse(
  error: unknown,
  defaultMessage: string = 'An unexpected error occurred'
): NextResponse {
  if (error instanceof ApiError) {
    const response: ApiErrorResponse = {
      error: error.message,
      code: error.code,
      details: error.details,
      context: error.context,
      timestamp: new Date().toISOString()
    }
    return NextResponse.json(response, { status: error.statusCode })
  }

  if (error instanceof z.ZodError) {
    return createErrorResponse(new ValidationError('Invalid input data', error))
  }

  // Handle generic errors
  console.error('Unhandled error in API:', error)
  
  const response: ApiErrorResponse = {
    error: defaultMessage,
    code: 'INTERNAL_ERROR',
    details: error instanceof Error ? error.message : 'Unknown error',
    timestamp: new Date().toISOString()
  }
  
  return NextResponse.json(response, { status: 500 })
}

export function createSuccessResponse<T>(
  data: T,
  message?: string,
  additionalFields?: Record<string, any>
): NextResponse {
  const response: ApiSuccessResponse<T> = {
    success: true,
    data,
    ...(message && { message }),
    ...additionalFields
  }
  
  return NextResponse.json(response)
}

export function handleKubernetesOperation<T>(
  operation: string,
  promise: Promise<T>
): Promise<T> {
  return promise.catch((error) => {
    throw new KubernetesError(operation, error)
  })
}

export function validateClusterNameFormat(clusterName: string): void {
  if (!clusterName) {
    throw new InvalidClusterNameError('', 'Cluster name is required')
  }

  // Kubernetes name validation: lowercase alphanumeric with hyphens
  // Cannot start or end with hyphen, max 253 characters
  const nameRegex = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/
  const maxLength = 253

  if (clusterName.length > maxLength) {
    throw new InvalidClusterNameError(
      clusterName, 
      `Cluster name must be ${maxLength} characters or less`
    )
  }

  if (!nameRegex.test(clusterName)) {
    throw new InvalidClusterNameError(
      clusterName,
      'Cluster name must be lowercase letters, numbers, and hyphens only. Cannot start or end with hyphen.'
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

export function createPermissionDeniedError(
  action: string,
  resource: string,
  userRole?: string
): ApiError {
  return new ApiError(
    `Permission denied: ${action}`,
    'PERMISSION_DENIED',
    403,
    `You don't have permission to ${action} ${resource}`,
    { action, resource, userRole }
  )
}

export function createAuthenticationRequiredError(): ApiError {
  return new ApiError(
    'Authentication required',
    'AUTHENTICATION_REQUIRED',
    401,
    'You must be logged in to access this resource'
  )
}

export function createResourceConflictError(
  resourceType: string,
  resourceName: string,
  reason: string
): ApiError {
  return new ApiError(
    `${resourceType} '${resourceName}' conflict`,
    'RESOURCE_CONFLICT',
    409,
    reason,
    { resourceType, resourceName }
  )
}

