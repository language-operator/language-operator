import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import { getUserOrganization } from '@/lib/organization-context'
import { validateClusterExists } from '@/lib/cluster-validation'
import {
  createErrorResponse,
  handleKubernetesOperation,
  validateClusterNameFormat,
} from '@/lib/api-error-handler'
import yaml from 'js-yaml'

// GET /api/clusters/[name]/personas/[personaName]/yaml - Get persona YAML
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    // Get user's selected organization (replaces broken memberships[0] pattern)
    const { user, organization, userRole } = await getUserOrganization(request)
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, personaName } = await params

    // Validate cluster name format
    validateClusterNameFormat(clusterName)

    // Validate cluster exists and user has access
    await validateClusterExists(organization.namespace, clusterName, { validateAccess: true })

    // Fetch specific persona from organization namespace
    const response = await handleKubernetesOperation(
      'get persona for YAML',
      k8sClient.getLanguagePersona(organization.namespace, personaName)
    )

    // Handle different response structures from k8s client
    let persona: any = null
    if ((response as any)?.body) {
      persona = (response as any).body
    } else if ((response as any)?.data) {
      persona = (response as any).data
    } else if (response) {
      persona = response as any
    }

    if (!persona) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' not found`),
        'Persona not found'
      )
    }

    // Verify the persona belongs to this cluster (check clusterRef)
    if (persona.spec?.clusterRef !== clusterName) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' does not belong to cluster '${clusterName}'`),
        'Persona not found in cluster'
      )
    }

    // Verify persona belongs to user's organization
    const personaOrgLabel = persona.metadata?.labels?.['langop.io/organization-id']
    if (personaOrgLabel && personaOrgLabel !== organization.id) {
      return createErrorResponse(
        new Error(`Persona '${personaName}' not found`),
        'Persona not found'
      )
    }

    // Convert persona object to YAML
    const yamlContent = yaml.dump(persona, {
      indent: 2,
      lineWidth: 120,
      quotingType: '"',
      forceQuotes: false,
    })

    // Return as plain text with appropriate content-type
    return new NextResponse(yamlContent, {
      headers: {
        'Content-Type': 'text/yaml; charset=utf-8',
        'Cache-Control': 'no-cache',
      },
    })

  } catch (error) {
    console.error('Error fetching persona YAML:', error)
    return createErrorResponse(error, 'Failed to fetch persona YAML')
  }
}