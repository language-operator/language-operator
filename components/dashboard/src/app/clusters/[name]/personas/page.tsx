'use client'

import { useParams } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Users, Plus } from 'lucide-react'
import Link from 'next/link'

export default function ClusterPersonas() {
  const params = useParams()
  const clusterName = params?.name as string

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Personas</h1>
            <p className="text-gray-600 mt-1">
              AI personas configured for the {clusterName} cluster
            </p>
          </div>
          <Button asChild>
            <Link href={`/clusters/${clusterName}/personas/new`}>
              <Plus className="h-4 w-4 mr-2" />
              New Persona
            </Link>
          </Button>
        </div>

        {/* Empty State */}
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Users className="h-16 w-16 text-gray-400 mb-4" />
            <CardTitle className="text-xl mb-2">No personas yet</CardTitle>
            <CardDescription className="text-center max-w-md mb-6">
              Personas define the behavior, knowledge, and communication style 
              for AI agents. Create your first persona to get started.
            </CardDescription>
            <Button asChild>
              <Link href={`/clusters/${clusterName}/personas/new`}>
                <Plus className="h-4 w-4 mr-2" />
                Create Your First Persona
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}