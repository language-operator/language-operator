import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'

export async function GET(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session || !session.activeOrganization) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const k8sClient = getKubernetesClient()
    const namespace = session.activeOrganization.namespace
    
    const agents = await k8sClient.getLanguageAgents(namespace)
    
    return NextResponse.json({ agents })
  } catch (error) {
    console.error('Error fetching agents:', error)
    return NextResponse.json(
      { error: 'Failed to fetch agents' }, 
      { status: 500 }
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    const session = await getServerSession(authOptions)
    
    if (!session || !session.activeOrganization) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const body = await request.json()
    const k8sClient = getKubernetesClient()
    const namespace = session.activeOrganization.namespace
    
    const agent = await k8sClient.createLanguageAgent(namespace, body)
    
    return NextResponse.json({ agent })
  } catch (error) {
    console.error('Error creating agent:', error)
    return NextResponse.json(
      { error: 'Failed to create agent' }, 
      { status: 500 }
    )
  }
}