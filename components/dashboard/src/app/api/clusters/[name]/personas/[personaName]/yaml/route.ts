import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { k8sClient } from '@/lib/k8s-client'
import { db } from '@/lib/db'
import { requirePermission } from '@/lib/permissions'
import yaml from 'js-yaml'

// GET /api/clusters/[name]/personas/[personaName]/yaml - Get persona YAML
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; personaName: string }> }
) {
  try {
    const session = await getServerSession(authOptions)
    if (!session?.user?.email) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const user = await db.user.findUnique({
      where: { email: session.user.email },
      include: { memberships: { include: { organization: true } } },
    })

    if (!user || user.memberships.length === 0) {
      return NextResponse.json({ error: 'No organization found' }, { status: 404 })
    }

    const organization = user.memberships[0].organization
    
    const hasPermission = await requirePermission(user.id, organization.id, 'view')
    if (!hasPermission) {
      return NextResponse.json({ error: 'Insufficient permissions' }, { status: 403 })
    }

    const { name: clusterName, personaName } = await params
    if (!clusterName || !personaName) {
      return NextResponse.json({ error: 'Cluster name and persona name are required' }, { status: 400 })
    }

    // Fetch specific persona from organization namespace
    let persona: any = null
    
    try {
      const response = await k8sClient.getLanguagePersona(organization.namespace, personaName)
      
      // Handle different response structures from k8s client
      if ((response as any)?.body) {
        persona = (response as any).body
      } else if ((response as any)?.data) {
        persona = (response as any).data
      } else if (response) {
        persona = response as any
      }
    } catch (k8sError) {
      // If persona not found, return 404
      if (k8sError instanceof Error && k8sError.message.includes('404')) {
        return NextResponse.json({ 
          error: 'Persona not found',
          details: `Persona "${personaName}" not found in cluster "${clusterName}"` 
        }, { status: 404 })
      }
      
      console.error('Error fetching persona from Kubernetes:', k8sError)
      throw k8sError
    }

    if (!persona) {
      return NextResponse.json({ 
        error: 'Persona not found',
        details: `Persona "${personaName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
    }

    // Verify persona belongs to user's organization
    const personaOrgLabel = persona.metadata?.labels?.['langop.io/organization-id']
    if (personaOrgLabel && personaOrgLabel !== organization.id) {
      return NextResponse.json({ 
        error: 'Persona not found',
        details: `Persona "${personaName}" not found in cluster "${clusterName}"` 
      }, { status: 404 })
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
    return NextResponse.json({ 
      error: 'Failed to fetch persona YAML',
      details: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}