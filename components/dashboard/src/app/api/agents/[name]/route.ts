import { NextRequest, NextResponse } from 'next/server'
import { getServerSession } from 'next-auth'
import { authOptions } from '@/lib/auth'
import { getKubernetesClient } from '@/lib/kubernetes'

export async function PATCH(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session || !session.activeOrganization) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const body = await request.json()
    const k8sClient = getKubernetesClient()
    const namespace = session.activeOrganization.namespace
    
    const agent = await k8sClient.updateLanguageAgent(namespace, name, body)
    
    return NextResponse.json({ agent })
  } catch (error) {
    console.error('Error updating agent:', error)
    return NextResponse.json(
      { error: 'Failed to update agent' }, 
      { status: 500 }
    )
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ name: string }> }
) {
  try {
    const { name } = await params
    const session = await getServerSession(authOptions)
    
    if (!session || !session.activeOrganization) {
      return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }

    const k8sClient = getKubernetesClient()
    const namespace = session.activeOrganization.namespace
    
    await k8sClient.deleteLanguageAgent(namespace, name)
    
    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting agent:', error)
    return NextResponse.json(
      { error: 'Failed to delete agent' }, 
      { status: 500 }
    )
  }
}