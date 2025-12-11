'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

// Organization form schema matching the API validation
const organizationFormSchema = z.object({
  name: z.string().min(1, 'Organization name is required').max(100, 'Name must be 100 characters or less'),
  slug: z.string()
    .min(1, 'Slug is required')
    .max(50, 'Slug must be 50 characters or less')
    .regex(/^[a-z0-9-]+$/, 'Slug must contain only lowercase letters, numbers, and hyphens'),
  namespace: z.string()
    .min(1, 'Namespace is required')
    .max(63, 'Namespace must be 63 characters or less')
    .regex(/^[a-z0-9-]+$/, 'Namespace must contain only lowercase letters, numbers, and hyphens'),
  plan: z.enum(['free', 'pro', 'enterprise'], 'Please select a plan')
})

export type OrganizationFormData = z.infer<typeof organizationFormSchema>

interface OrganizationFormProps {
  initialData?: Partial<OrganizationFormData>
  onSubmit: (data: OrganizationFormData) => void
  onCancel: () => void
  isLoading?: boolean
  mode: 'create' | 'edit'
}

function generateSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '') // Remove invalid characters
    .replace(/\s+/g, '-') // Replace spaces with hyphens
    .replace(/-+/g, '-') // Replace multiple hyphens with single
    .replace(/^-|-$/g, '') // Remove leading/trailing hyphens
}

export function OrganizationForm({
  initialData,
  onSubmit,
  onCancel,
  isLoading = false,
  mode
}: OrganizationFormProps) {
  const [isAutoGenerating, setIsAutoGenerating] = useState(mode === 'create')

  const form = useForm<OrganizationFormData>({
    resolver: zodResolver(organizationFormSchema),
    defaultValues: {
      name: initialData?.name || '',
      slug: initialData?.slug || '',
      namespace: initialData?.namespace || '',
      plan: initialData?.plan || 'free'
    }
  })

  const { watch, setValue, formState: { errors } } = form
  const nameValue = watch('name')
  const slugValue = watch('slug')

  // Auto-generate slug and namespace when name changes (only in create mode or if user hasn't manually edited)
  useEffect(() => {
    if (mode === 'create' && isAutoGenerating && nameValue) {
      const newSlug = generateSlug(nameValue)
      setValue('slug', newSlug)
      setValue('namespace', newSlug)
    }
  }, [nameValue, setValue, mode, isAutoGenerating])

  // Auto-generate namespace when slug changes (if auto-generating)
  useEffect(() => {
    if (isAutoGenerating && slugValue) {
      setValue('namespace', slugValue)
    }
  }, [slugValue, setValue, isAutoGenerating])

  const handleSlugChange = (value: string) => {
    // If user manually edits slug, disable auto-generation
    setIsAutoGenerating(false)
    setValue('slug', value)
  }

  const handleNamespaceChange = (value: string) => {
    // If user manually edits namespace, disable auto-generation
    setIsAutoGenerating(false)
    setValue('namespace', value)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Organization Name</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder="My Organization"
                  disabled={isLoading}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="slug"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Slug</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder="my-organization"
                  disabled={isLoading}
                  onChange={(e) => handleSlugChange(e.target.value)}
                />
              </FormControl>
              <FormMessage />
              {mode === 'create' && (
                <p className="text-xs text-gray-600">
                  Used in URLs and API endpoints. {isAutoGenerating ? 'Auto-generated from name.' : 'Edit to customize.'}
                </p>
              )}
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="namespace"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Kubernetes Namespace</FormLabel>
              <FormControl>
                <Input
                  {...field}
                  placeholder="my-organization"
                  disabled={isLoading}
                  onChange={(e) => handleNamespaceChange(e.target.value)}
                />
              </FormControl>
              <FormMessage />
              <p className="text-xs text-gray-600">
                Kubernetes namespace for this organization's resources. {isAutoGenerating ? 'Auto-generated from slug.' : 'Edit to customize.'}
              </p>
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="plan"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Plan</FormLabel>
              <Select
                onValueChange={field.onChange}
                defaultValue={field.value}
                disabled={isLoading}
              >
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a plan" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="free">Free</SelectItem>
                  <SelectItem value="pro">Pro</SelectItem>
                  <SelectItem value="enterprise">Enterprise</SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="flex justify-end gap-3 pt-4">
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? (
              <>
                {mode === 'create' ? 'Creating...' : 'Saving...'}
              </>
            ) : (
              <>
                {mode === 'create' ? 'Create Organization' : 'Save Changes'}
              </>
            )}
          </Button>
        </div>
      </form>
    </Form>
  )
}