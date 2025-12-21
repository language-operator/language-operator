'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'sonner'
import { useOrganizations } from '@/hooks/use-organizations'
import { fetchWithOrganization } from '@/lib/api-client'
import type { Organization } from '@/store/organization-store'
import type { OrganizationQuota, QUOTA_DESCRIPTIONS } from '@/types/quota'
import { QUOTA_LABELS } from '@/types/quota'

// Form schema
const quotaFormSchema = z.object({
  'count/languageagents': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagemodels': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagetools': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagepersonas': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languageclusters': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'requests.cpu': z.string().regex(/^\d+(m)?$/, 'Must be in format: 100m or 1'),
  'requests.memory': z.string().regex(/^\d+(Ki|Mi|Gi)$/, 'Must be in format: 128Mi or 2Gi'),
  'limits.cpu': z.string().regex(/^\d+(m)?$/, 'Must be in format: 100m or 1'),
  'limits.memory': z.string().regex(/^\d+(Ki|Mi|Gi)$/, 'Must be in format: 128Mi or 2Gi'),
})

type QuotaFormValues = z.infer<typeof quotaFormSchema>

interface QuotaData {
  quota: OrganizationQuota
  used: Partial<OrganizationQuota>
  available: Partial<OrganizationQuota>
  percentUsed: Record<string, number>
}

export default function EditOrganizationPage() {
  const router = useRouter()
  const params = useParams()
  const organizationId = params.id as string

  const [isLoadingQuota, setIsLoadingQuota] = useState(false)
  const [quotaData, setQuotaData] = useState<QuotaData | null>(null)
  const [isUpdating, setIsUpdating] = useState(false)
  const [organization, setOrganization] = useState<Organization | null>(null)
  const [isLoadingOrg, setIsLoadingOrg] = useState(true)
  const [userRole, setUserRole] = useState<string | null>(null)

  // Quota form
  const quotaForm = useForm<QuotaFormValues>({
    resolver: zodResolver(quotaFormSchema),
    defaultValues: {
      'count/languageagents': '2',
      'count/languagemodels': '2',
      'count/languagetools': '5',
      'count/languagepersonas': '3',
      'count/languageclusters': '1',
      'requests.cpu': '1000m',
      'requests.memory': '2Gi',
      'limits.cpu': '2000m',
      'limits.memory': '4Gi'
    }
  })

  // Load organization data
  useEffect(() => {
    const fetchOrganization = async () => {
      setIsLoadingOrg(true)
      try {
        const response = await fetch(`/api/organizations/${organizationId}`)
        if (!response.ok) throw new Error('Failed to fetch organization')

        const data = await response.json()
        setOrganization(data.organization)
        setUserRole(data.userRole)
      } catch (error) {
        console.error('Error fetching organization:', error)
        toast.error('Failed to load organization')
      } finally {
        setIsLoadingOrg(false)
      }
    }

    fetchOrganization()
  }, [organizationId])

  // Load quota data
  useEffect(() => {
    if (!organizationId) return

    const fetchQuotaData = async () => {
      setIsLoadingQuota(true)
      try {
        const response = await fetch(`/api/organizations/${organizationId}/quota`)
        if (!response.ok) throw new Error('Failed to fetch quota data')

        const data = await response.json()
        setQuotaData(data.data)

        // Populate quota form with current values
        if (data.data.quota) {
          quotaForm.reset(data.data.quota)
        }
      } catch (error) {
        console.error('Error fetching quota data:', error)
        toast.error('Failed to load quota data')
      } finally {
        setIsLoadingQuota(false)
      }
    }

    fetchQuotaData()
  }, [organizationId])

  // Handle quota form submission
  const onQuotaSubmit = async (values: QuotaFormValues) => {
    setIsUpdating(true)
    try {
      const response = await fetch(`/api/organizations/${organizationId}/quota`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ quotas: values })
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Failed to update quotas')
      }

      const data = await response.json()
      setQuotaData(data.data)

      toast.success('Quotas updated successfully')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to update quotas')
    } finally {
      setIsUpdating(false)
    }
  }

  const handleCancel = () => {
    router.push('/settings/organizations')
  }

  // Check permissions
  const canEditQuotas = userRole === 'owner' || userRole === 'admin'

  if (isLoadingOrg) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  if (!organization) {
    return (
      <div className="text-center py-12">
        <h3 className="text-lg font-medium">Organization not found</h3>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {!canEditQuotas && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <p className="text-sm text-yellow-800">
            You need owner or admin permissions to edit resource quotas.
          </p>
        </div>
      )}

              <Form {...quotaForm}>
                <form onSubmit={quotaForm.handleSubmit(onQuotaSubmit)} className="space-y-6">
                  {/* Resource Count Quotas */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Resource Counts</CardTitle>
                      <CardDescription>
                        Maximum number of resources this organization can create
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <QuotaField
                        form={quotaForm}
                        name="count/languageagents"
                        label={QUOTA_LABELS['count/languageagents']}
                        description="Maximum number of agents"
                        currentUsed={quotaData?.used?.['count/languageagents'] || '0'}
                        disabled={!canEditQuotas || isLoadingQuota}
                      />
                      <QuotaField
                        form={quotaForm}
                        name="count/languagemodels"
                        label={QUOTA_LABELS['count/languagemodels']}
                        description="Maximum number of model configurations"
                        currentUsed={quotaData?.used?.['count/languagemodels'] || '0'}
                        disabled={!canEditQuotas || isLoadingQuota}
                      />
                      <QuotaField
                        form={quotaForm}
                        name="count/languagetools"
                        label={QUOTA_LABELS['count/languagetools']}
                        description="Maximum number of tools"
                        currentUsed={quotaData?.used?.['count/languagetools'] || '0'}
                        disabled={!canEditQuotas || isLoadingQuota}
                      />
                      <QuotaField
                        form={quotaForm}
                        name="count/languagepersonas"
                        label={QUOTA_LABELS['count/languagepersonas']}
                        description="Maximum number of personas"
                        currentUsed={quotaData?.used?.['count/languagepersonas'] || '0'}
                        disabled={!canEditQuotas || isLoadingQuota}
                      />
                      <QuotaField
                        form={quotaForm}
                        name="count/languageclusters"
                        label={QUOTA_LABELS['count/languageclusters']}
                        description="Maximum number of clusters"
                        currentUsed={quotaData?.used?.['count/languageclusters'] || '0'}
                        disabled={!canEditQuotas || isLoadingQuota}
                      />
                    </CardContent>
                  </Card>

                  {/* CPU & Memory Quotas */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Compute Resources</CardTitle>
                      <CardDescription>
                        CPU and memory limits for all resources in this organization
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid grid-cols-2 gap-4">
                        <QuotaField
                          form={quotaForm}
                          name="requests.cpu"
                          label={QUOTA_LABELS['requests.cpu']}
                          description="e.g., 1000m or 1"
                          currentUsed={quotaData?.used?.['requests.cpu'] || '0'}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <QuotaField
                          form={quotaForm}
                          name="limits.cpu"
                          label={QUOTA_LABELS['limits.cpu']}
                          description="e.g., 2000m or 2"
                          currentUsed={quotaData?.used?.['limits.cpu'] || '0'}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <QuotaField
                          form={quotaForm}
                          name="requests.memory"
                          label={QUOTA_LABELS['requests.memory']}
                          description="e.g., 2Gi or 1024Mi"
                          currentUsed={quotaData?.used?.['requests.memory'] || '0'}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <QuotaField
                          form={quotaForm}
                          name="limits.memory"
                          label={QUOTA_LABELS['limits.memory']}
                          description="e.g., 4Gi or 2048Mi"
                          currentUsed={quotaData?.used?.['limits.memory'] || '0'}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                      </div>
                    </CardContent>
                  </Card>

                  <div className="flex justify-end space-x-4">
                    <Button type="button" variant="outline" onClick={handleCancel}>
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      disabled={isUpdating || !canEditQuotas || isLoadingQuota}
                    >
                      {isUpdating ? 'Updating...' : 'Update Quotas'}
                    </Button>
                  </div>
                </form>
              </Form>
    </div>
  )
}

// Helper component for quota fields
function QuotaField({
  form,
  name,
  label,
  description,
  currentUsed,
  disabled
}: {
  form: any
  name: string
  label: string
  description: string
  currentUsed: string
  disabled: boolean
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <div className="flex items-center gap-2">
            <FormControl>
              <Input {...field} disabled={disabled} className="max-w-xs" />
            </FormControl>
            <span className="text-sm text-muted-foreground">
              (used: {currentUsed})
            </span>
          </div>
          <FormDescription>{description}</FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
