'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { uniqueNamesGenerator, adjectives, animals } from 'unique-names-generator'

// Organization form schema matching the API validation
const organizationFormSchema = z.object({
  name: z.string().min(1, 'Organization name is required').max(100, 'Name must be 100 characters or less'),
  switchToNew: z.boolean()
})

export type OrganizationFormData = z.infer<typeof organizationFormSchema>

interface OrganizationFormProps {
  initialData?: Partial<OrganizationFormData>
  onSubmit: (data: OrganizationFormData) => void
  onCancel: () => void
  isLoading?: boolean
  mode: 'create' | 'edit'
}

const nameConfig = {
  dictionaries: [adjectives, animals],
  separator: ' ',
  style: 'capital' as const
}

export function OrganizationForm({
  initialData,
  onSubmit,
  onCancel,
  isLoading = false,
  mode
}: OrganizationFormProps) {
  const [funnyName, setFunnyName] = useState('')

  const form = useForm<OrganizationFormData>({
    resolver: zodResolver(organizationFormSchema),
    defaultValues: {
      name: initialData?.name || '',
      switchToNew: false
    }
  })

  const { watch, setValue, formState: { errors } } = form

  // Generate a funny default name on component mount
  useEffect(() => {
    if (mode === 'create' && !initialData?.name) {
      const generatedName = uniqueNamesGenerator(nameConfig)
      setFunnyName(generatedName)
      setValue('name', generatedName)
    }
  }, [mode, initialData?.name, setValue])

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <Input
                  {...field}
                  disabled={isLoading}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="switchToNew"
          render={({ field }) => (
            <FormItem className="flex flex-row items-start space-x-3 space-y-0">
              <FormControl>
                <Checkbox
                  checked={field.value}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
              <div className="space-y-1 leading-none">
                <FormLabel className="text-sm">
                  Switch to new organization
                </FormLabel>
              </div>
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