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
    
    const models = await k8sClient.getLanguageModels(namespace)
    
    return NextResponse.json({ models })
  } catch (error) {
    console.error('Error fetching models:', error)
    return NextResponse.json(
      { error: 'Failed to fetch models' }, 
      { status: 500 }
    )
  }
}