'use client'

import { useState, useEffect, useRef } from 'react'
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
import { useOrganization } from '@/components/organization-provider'
import type { OrganizationQuota, QUOTA_DESCRIPTIONS } from '@/types/quota'
import { QUOTA_LABELS } from '@/types/quota'
import { Boxes, Bot, Wrench, Users, Cpu, UserPlus, Plus, Minus } from 'lucide-react'

// Form schema
const quotaFormSchema = z.object({
  'count/languageagents': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagemodels': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagetools': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languagepersonas': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/languageclusters': z.string().regex(/^\d+$/, 'Must be a positive integer'),
  'count/members': z.string().regex(/^\d+$/, 'Must be a positive integer'),
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
  const { getOrgUrl } = useOrganization()

  const [isLoadingQuota, setIsLoadingQuota] = useState(true)
  const [quotaData, setQuotaData] = useState<QuotaData | null>(null)
  const [isUpdating, setIsUpdating] = useState(false)
  const [organization, setOrganization] = useState<Organization | null>(null)
  const [isLoadingOrg, setIsLoadingOrg] = useState(true)
  const [userRole, setUserRole] = useState<string | null>(null)
  const hasLoadedInitialQuota = useRef(false)

  // Quota form
  const quotaForm = useForm<QuotaFormValues>({
    resolver: zodResolver(quotaFormSchema),
    defaultValues: {
      'count/languageagents': '2',
      'count/languagemodels': '2',
      'count/languagetools': '5',
      'count/languagepersonas': '3',
      'count/languageclusters': '1',
      'count/members': '5',
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

        // Only populate quota form on initial load, not after updates
        if (data.data.quota && !hasLoadedInitialQuota.current) {
          console.log('[QUOTA LOAD] API quota data:', JSON.stringify(data.data.quota))
          // Use setValue for each field instead of reset() due to react-hook-form
          // limitations with dot-notation field names
          const quota = data.data.quota
          const setFormValue = quotaForm.setValue as any
          if (quota['count/languageagents']) setFormValue('count/languageagents', String(quota['count/languageagents']))
          if (quota['count/languagemodels']) setFormValue('count/languagemodels', String(quota['count/languagemodels']))
          if (quota['count/languagetools']) setFormValue('count/languagetools', String(quota['count/languagetools']))
          if (quota['count/languagepersonas']) setFormValue('count/languagepersonas', String(quota['count/languagepersonas']))
          if (quota['count/languageclusters']) setFormValue('count/languageclusters', String(quota['count/languageclusters']))
          if (quota['count/members']) setFormValue('count/members', String(quota['count/members']))
          if (quota['requests.cpu']) setFormValue('requests.cpu', String(quota['requests.cpu']))
          if (quota['requests.memory']) setFormValue('requests.memory', String(quota['requests.memory']))
          if (quota['limits.cpu']) setFormValue('limits.cpu', String(quota['limits.cpu']))
          if (quota['limits.memory']) setFormValue('limits.memory', String(quota['limits.memory']))
          hasLoadedInitialQuota.current = true
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
  const onQuotaSubmit = async () => {
    // Build values object by manually reading each field using watch()
    // This works around a react-hook-form limitation with dot-notation field names
    // where getValues() doesn't properly track changes to fields with dots in their names
    const values = {
      'count/languageagents': quotaForm.watch('count/languageagents'),
      'count/languagemodels': quotaForm.watch('count/languagemodels'),
      'count/languagetools': quotaForm.watch('count/languagetools'),
      'count/languagepersonas': quotaForm.watch('count/languagepersonas'),
      'count/languageclusters': quotaForm.watch('count/languageclusters'),
      'count/members': quotaForm.watch('count/members'),
      'requests.cpu': quotaForm.watch('requests.cpu'),
      'requests.memory': quotaForm.watch('requests.memory'),
      'limits.cpu': quotaForm.watch('limits.cpu'),
      'limits.memory': quotaForm.watch('limits.memory'),
    }
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

      // Don't reset the form - it already has the correct values the user just submitted
      // The API returns current k8s state which might not reflect the update yet

      toast.success('Quotas updated successfully')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to update quotas')
    } finally {
      setIsUpdating(false)
    }
  }

  const handleCancel = () => {
    router.push(getOrgUrl('/settings/organizations'))
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
                    <CardContent>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        <ResourceCountField
                          form={quotaForm}
                          name="count/languageclusters"
                          label="Clusters"
                          icon={Boxes}
                          currentUsed={quotaData?.used?.['count/languageclusters']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <ResourceCountField
                          form={quotaForm}
                          name="count/languageagents"
                          label="Agents"
                          icon={Bot}
                          currentUsed={quotaData?.used?.['count/languageagents']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <ResourceCountField
                          form={quotaForm}
                          name="count/languagetools"
                          label="Tools"
                          icon={Wrench}
                          currentUsed={quotaData?.used?.['count/languagetools']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <ResourceCountField
                          form={quotaForm}
                          name="count/languagepersonas"
                          label="Personas"
                          icon={Users}
                          currentUsed={quotaData?.used?.['count/languagepersonas']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <ResourceCountField
                          form={quotaForm}
                          name="count/languagemodels"
                          label="Models"
                          icon={Cpu}
                          currentUsed={quotaData?.used?.['count/languagemodels']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <ResourceCountField
                          form={quotaForm}
                          name="count/members"
                          label="Members"
                          icon={UserPlus}
                          currentUsed={quotaData?.used?.['count/members']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                      </div>
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
                          currentUsed={quotaData?.used?.['requests.cpu']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <QuotaField
                          form={quotaForm}
                          name="limits.cpu"
                          label={QUOTA_LABELS['limits.cpu']}
                          currentUsed={quotaData?.used?.['limits.cpu']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-4">
                        <QuotaField
                          form={quotaForm}
                          name="requests.memory"
                          label={QUOTA_LABELS['requests.memory']}
                          currentUsed={quotaData?.used?.['requests.memory']}
                          disabled={!canEditQuotas || isLoadingQuota}
                        />
                        <QuotaField
                          form={quotaForm}
                          name="limits.memory"
                          label={QUOTA_LABELS['limits.memory']}
                          currentUsed={quotaData?.used?.['limits.memory']}
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

// Helper component for resource count fields with icons and +/- buttons
function ResourceCountField({
  form,
  name,
  label,
  icon: Icon,
  currentUsed,
  disabled
}: {
  form: any
  name: string
  label: string
  icon: any
  currentUsed?: string
  disabled: boolean
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => {
        const currentValue = parseInt(field.value) || 0

        const increment = () => {
          field.onChange(String(currentValue + 1))
        }

        const decrement = () => {
          if (currentValue > 0) {
            field.onChange(String(currentValue - 1))
          }
        }

        return (
          <FormItem>
            <div className="border rounded-lg p-4 space-y-3">
              <div className="flex items-center justify-between">
                <FormLabel className="text-base font-medium">{label}</FormLabel>
                <Icon className="w-5 h-5 text-muted-foreground" />
              </div>

              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={decrement}
                  disabled={disabled || currentValue <= 0}
                  className="h-10 w-10"
                >
                  <Minus className="h-4 w-4" />
                </Button>

                <FormControl>
                  <Input
                    {...field}
                    disabled={disabled}
                    className="text-center font-bold h-16 max-w-[120px]"
                    style={{ fontSize: '1.75rem' }}
                  />
                </FormControl>

                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  onClick={increment}
                  disabled={disabled}
                  className="h-10 w-10"
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </div>

              {currentUsed && (
                <FormDescription className="text-xs text-center">
                  Currently used: {currentUsed}
                </FormDescription>
              )}

              <FormMessage />
            </div>
          </FormItem>
        )
      }}
    />
  )
}

// Helper component for quota fields
function QuotaField({
  form,
  name,
  label,
  currentUsed,
  disabled
}: {
  form: any
  name: string
  label: string
  currentUsed?: string
  disabled: boolean
}) {
  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => {
        console.log(`[FIELD ${name}] Current value:`, field.value)
        return (
          <FormItem>
            <FormLabel>{label}</FormLabel>
            <FormControl>
              <Input
                {...field}
                disabled={disabled}
                className="max-w-xs"
              />
            </FormControl>
            {currentUsed && (
              <FormDescription>
                Currently used: {currentUsed}
              </FormDescription>
            )}
            <FormMessage />
          </FormItem>
        )
      }}
    />
  )
}
