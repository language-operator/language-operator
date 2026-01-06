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
  // Organization-specific error codes
  | 'ORGANIZATION_NOT_FOUND'
  | 'ORGANIZATION_MISMATCH'
  | 'ORGANIZATION_ACCESS_DENIED'
  | 'INVALID_ORGANIZATION_ID'
  | 'ORGANIZATION_CONTEXT_MISSING'
  | 'RATE_LIMITED'
  | 'INSUFFICIENT_PERMISSIONS'
  // Persona generation error codes
  | 'MODEL_NOT_AVAILABLE'
  | 'MODEL_ENDPOINT_ERROR'
  | 'MODEL_RESPONSE_ERROR'
  | 'GENERATION_PARSING_ERROR'
  | 'GENERATION_TIMEOUT'

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

// Organization-specific error helpers
export class OrganizationNotFoundError extends ApiError {
  constructor(organizationId: string) {
    super(
      `Organization '${organizationId}' not found`,
      'ORGANIZATION_NOT_FOUND',
      404,
      `The organization '${organizationId}' does not exist or you don't have access to it`,
      { organizationId }
    )
  }
}

export class OrganizationMismatchError extends ApiError {
  constructor(urlOrgId: string, contextOrgId?: string) {
    super(
      'Organization ID mismatch',
      'ORGANIZATION_MISMATCH',
      400,
      'Organization ID in URL does not match the organization context',
      { urlOrgId, contextOrgId }
    )
  }
}

export class OrganizationAccessDeniedError extends ApiError {
  constructor(organizationId: string, userId?: string) {
    super(
      `Access denied to organization '${organizationId}'`,
      'ORGANIZATION_ACCESS_DENIED',
      403,
      `You don't have permission to access organization '${organizationId}'`,
      { organizationId, userId }
    )
  }
}

export class InvalidOrganizationIdError extends ApiError {
  constructor(organizationId: string) {
    super(
      `Invalid organization ID '${organizationId}'`,
      'INVALID_ORGANIZATION_ID',
      400,
      'Organization ID must be a valid UUID or CUID',
      { organizationId }
    )
  }
}

export class OrganizationContextMissingError extends ApiError {
  constructor() {
    super(
      'Organization context not found',
      'ORGANIZATION_CONTEXT_MISSING',
      400,
      'Organization context was not set by middleware. This may indicate a configuration issue.'
    )
  }
}

export class RateLimitError extends ApiError {
  constructor(userId: string, organizationId?: string) {
    super(
      'Too many requests',
      'RATE_LIMITED',
      429,
      'Too many failed access attempts. Please try again later.',
      { userId, organizationId }
    )
  }
}

export class InsufficientPermissionsError extends ApiError {
  constructor(action: string, resource: string, userRole?: string, organizationId?: string) {
    super(
      `Insufficient permissions to ${action} ${resource}`,
      'INSUFFICIENT_PERMISSIONS', 
      403,
      `Your current role does not allow you to ${action} ${resource}`,
      { action, resource, userRole, organizationId }
    )
  }
}

/**
 * Enhanced error response creator with organization context
 */
export function createOrganizationErrorResponse(
  error: unknown,
  organizationId?: string,
  userId?: string,
  defaultMessage: string = 'An unexpected error occurred'
): NextResponse {
  // Log security-relevant errors with organization context
  if (error instanceof ApiError && ['ORGANIZATION_ACCESS_DENIED', 'INSUFFICIENT_PERMISSIONS', 'RATE_LIMITED'].includes(error.code)) {
    console.warn(`[Security] ${error.code}: ${error.message}`, {
      organizationId: organizationId || error.context?.organizationId,
      userId: userId || error.context?.userId,
      timestamp: new Date().toISOString()
    })
  }

  return createErrorResponse(error, defaultMessage)
}

/**
 * Validate organization context helper for API routes
 */
export function validateOrganizationContext(
  urlOrgId: string,
  contextOrgId?: string | null,
  userId?: string
): void {
  if (!contextOrgId) {
    throw new OrganizationContextMissingError()
  }

  if (urlOrgId !== contextOrgId) {
    throw new OrganizationMismatchError(urlOrgId, contextOrgId)
  }
}

/**
 * Wrapper for org-aware API handlers with enhanced error handling
 */
export function withOrganizationErrorHandler<T extends any[], R>(
  handler: (...args: T) => Promise<Response>
) {
  return async (...args: T): Promise<Response> => {
    try {
      return await handler(...args)
    } catch (error) {
      return createOrganizationErrorResponse(error)
    }
  }
}

// Persona generation error classes
export class ModelNotAvailableError extends ApiError {
  constructor(modelName: string, namespace: string, status?: string) {
    const details = status === 'NotReady' 
      ? `Model '${modelName}' is not ready yet. Please wait for it to become available.`
      : `Model '${modelName}' was not found in namespace '${namespace}'. Please check if the model exists and is properly configured.`
    
    super(
      `Model '${modelName}' is not available`,
      'MODEL_NOT_AVAILABLE',
      404,
      details,
      { modelName, namespace, status }
    )
  }
}

export class ModelEndpointError extends ApiError {
  constructor(modelName: string, endpoint: string, cause?: string) {
    let details = `Unable to connect to model service at ${endpoint}.`
    
    if (cause?.includes('ECONNREFUSED')) {
      details += ' The model service may be down or not yet started.'
    } else if (cause?.includes('ETIMEDOUT')) {
      details += ' The request timed out. The model service may be overloaded.'
    } else if (cause?.includes('ENOTFOUND')) {
      details += ' The model endpoint could not be found. Check your cluster configuration.'
    } else {
      details += ' Please verify the model is running and accessible.'
    }
    
    super(
      `Cannot reach model '${modelName}'`,
      'MODEL_ENDPOINT_ERROR',
      503,
      details,
      { modelName, endpoint, cause }
    )
  }
}

export class ModelResponseError extends ApiError {
  constructor(modelName: string, statusCode: number, responseText?: string) {
    let details = `Model '${modelName}' returned an error response (${statusCode}).`
    
    if (statusCode === 400) {
      details += ' The request format may be invalid or unsupported by this model.'
    } else if (statusCode === 401 || statusCode === 403) {
      details += ' Authentication failed. Check if the model requires API keys.'
    } else if (statusCode === 429) {
      details += ' Rate limit exceeded. Please wait before trying again.'
    } else if (statusCode >= 500) {
      details += ' The model service is experiencing issues. Try again later.'
    }
    
    if (responseText) {
      details += ` Error details: ${responseText.slice(0, 200)}${responseText.length > 200 ? '...' : ''}`
    }
    
    super(
      `Model '${modelName}' error`,
      'MODEL_RESPONSE_ERROR',
      statusCode >= 500 ? 502 : 400,
      details,
      { modelName, statusCode, responseText }
    )
  }
}

export class GenerationParsingError extends ApiError {
  constructor(generatedText: string) {
    const preview = generatedText.slice(0, 100) + (generatedText.length > 100 ? '...' : '')
    
    super(
      'Invalid generation response format',
      'GENERATION_PARSING_ERROR',
      502,
      `The model generated text that could not be parsed as valid JSON. This usually means the model didn't follow the expected format. Generated text preview: "${preview}"`,
      { generatedText: preview }
    )
  }
}

export class GenerationTimeoutError extends ApiError {
  constructor(modelName: string, timeoutSeconds: number) {
    super(
      `Persona generation timed out`,
      'GENERATION_TIMEOUT',
      504,
      `Generation with model '${modelName}' timed out after ${timeoutSeconds} seconds. The model may be overloaded or the request too complex.`,
      { modelName, timeoutSeconds }
    )
  }
}

