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
    
    const personas = await k8sClient.getLanguagePersonas(namespace)
    
    return NextResponse.json({ personas })
  } catch (error) {
    console.error('Error fetching personas:', error)
    return NextResponse.json(
      { error: 'Failed to fetch personas' }, 
      { status: 500 }
    )
  }
}