// Generated TypeScript types for LanguageCluster CRD

import { V1ObjectMeta, V1Condition } from '@kubernetes/client-node'

// LanguageCluster CRD Types
export interface LanguageCluster {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageCluster'
  metadata: V1ObjectMeta
  spec: LanguageClusterSpec
  status?: LanguageClusterStatus
}

export interface LanguageClusterList {
  apiVersion: 'langop.io/v1alpha1'
  kind: 'LanguageClusterList'
  metadata: V1ObjectMeta
  items: LanguageCluster[]
}

export interface LanguageClusterSpec {
  // Domain configuration
  domain?: string
  
  // Ingress/Gateway configuration
  ingressConfig?: IngressConfig
}

export interface IngressConfig {
  // Gateway API configuration
  gatewayName?: string
  gatewayNamespace?: string
  gatewayClassName?: string // Deprecated, use gatewayName
  
  // Ingress fallback configuration
  ingressClassName?: string
  
  // TLS configuration
  tls?: TLSConfig
}

export interface TLSConfig {
  enabled?: boolean
  secretName?: string
  issuerRef?: IssuerReference
}

export interface IssuerReference {
  name: string
  kind?: 'Issuer' | 'ClusterIssuer'
  group?: string
}

// Status types
export interface LanguageClusterStatus {
  phase?: 'Pending' | 'Ready' | 'Failed' | 'Unknown'
  conditions?: V1Condition[]
}

// Frontend-specific types
export interface LanguageClusterFormData {
  name: string
  namespace: string
  
  // Domain configuration
  domain?: string
  
  // Gateway configuration
  gatewayName?: string
  gatewayNamespace?: string
  ingressClassName?: string
  
  // TLS configuration
  enableTLS?: boolean
  tlsSecretName?: string
  
  // Cert-manager configuration
  useCertManager?: boolean
  issuerName?: string
  issuerKind?: 'Issuer' | 'ClusterIssuer'
  issuerGroup?: string
}

export interface LanguageClusterListItem {
  name: string
  namespace: string
  domain?: string
  phase?: string
  age: string
  creationTimestamp: string
}

// API response types
export interface LanguageClusterResponse {
  success: boolean
  data?: LanguageCluster
  error?: string
}

export interface LanguageClusterListResponse {
  success: boolean
  data?: LanguageCluster[]
  error?: string
  total?: number
  page?: number
  limit?: number
}

// Query parameters for listing clusters
export interface LanguageClusterListParams {
  namespace?: string
  labelSelector?: string
  fieldSelector?: string
  page?: number
  limit?: number
  sortBy?: 'name' | 'domain' | 'phase' | 'age'
  sortOrder?: 'asc' | 'desc'
  search?: string
  phase?: string[]
  domain?: string
}