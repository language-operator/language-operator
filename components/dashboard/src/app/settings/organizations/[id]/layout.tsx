'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, Settings, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useOrganization } from '@/hooks/use-organizations'

interface OrganizationLayoutProps {
  children: React.ReactNode
}

export default function OrganizationLayout({ children }: OrganizationLayoutProps) {
  const params = useParams()
  const organizationId = params.id as string
  
  const { data: organizationData, isLoading } = useOrganization(organizationId)
  const organization = organizationData?.organization

  return (
    <div className="space-y-6">
      {/* Breadcrumb and Navigation */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="sm" asChild>
            <Link href="/settings/organizations">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Organizations
            </Link>
          </Button>
        </div>
        
        {organization && (
          <div>
            <h1 className="text-3xl font-bold">{organization.name}</h1>
            <p className="text-gray-600">
              Namespace: <code className="bg-gray-100 px-2 py-1 rounded text-sm">{organization.namespace}</code>
            </p>
          </div>
        )}
        
        {isLoading && (
          <div className="space-y-2">
            <div className="h-8 bg-gray-200 rounded w-1/4 animate-pulse"></div>
            <div className="h-4 bg-gray-200 rounded w-1/3 animate-pulse"></div>
          </div>
        )}
      </div>

      {/* Page Content */}
      {children}
    </div>
  )
}