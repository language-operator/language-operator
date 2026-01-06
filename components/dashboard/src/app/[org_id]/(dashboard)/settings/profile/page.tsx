'use client'

import React, { useEffect, useState } from 'react'
import { useSession } from 'next-auth/react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useToast } from '@/hooks/use-toast'
import { UserUpdateProfileSchema } from '@/lib/validation'
import { z } from 'zod'
import { fetchWithOrganization } from '@/lib/api-client'

export default function ProfilePage() {
  const { data: session, update: updateSession } = useSession()
  const [isEditing, setIsEditing] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    image: ''
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const { toast } = useToast()

  // Initialize form data when session loads or editing starts
  useEffect(() => {
    if (session?.user && isEditing) {
      setFormData({
        name: session.user.name || '',
        email: session.user.email || '',
        image: session.user.image || ''
      })
    }
  }, [session?.user, isEditing])

  const handleEditToggle = () => {
    setIsEditing(!isEditing)
    setErrors({})
    if (!isEditing && session?.user) {
      // Initialize form with current data when starting edit
      setFormData({
        name: session.user.name || '',
        email: session.user.email || '',
        image: session.user.image || ''
      })
    }
  }

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
    // Clear error for this field when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }))
    }
  }

  const handleSave = async () => {
    try {
      setIsLoading(true)
      setErrors({})

      // Validate form data
      const validation = UserUpdateProfileSchema.safeParse(formData)
      if (!validation.success) {
        const fieldErrors: Record<string, string> = {}
        if (validation.error?.issues) {
          validation.error.issues.forEach((error) => {
            if (error.path[0]) {
              fieldErrors[error.path[0] as string] = error.message
            }
          })
        }
        setErrors(fieldErrors)
        return
      }

      // Only send fields that have changed
      const updates: Record<string, string> = {}
      if (formData.name !== (session?.user?.name || '')) {
        updates.name = formData.name
      }
      if (formData.email !== (session?.user?.email || '')) {
        updates.email = formData.email
      }
      // Handle image specially - empty string should be sent to clear the image
      const currentImage = session?.user?.image || ''
      if (formData.image !== currentImage) {
        updates.image = formData.image
      }

      // If no changes, just close edit mode
      if (Object.keys(updates).length === 0) {
        setIsEditing(false)
        toast({
          title: 'No changes detected',
          description: 'Your profile is already up to date.',
        })
        return
      }

      // Send update request
      const response = await fetchWithOrganization('/api/profile', {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updates),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to update profile')
      }

      // Update session data
      await updateSession()
      
      setIsEditing(false)
      toast({
        title: 'Profile updated successfully',
        description: 'Your profile information has been saved.',
      })
    } catch (error) {
      console.error('Error updating profile:', error)
      toast({
        title: 'Error updating profile',
        description: error instanceof Error ? error.message : 'Please try again.',
        variant: 'destructive',
      })
    } finally {
      setIsLoading(false)
    }
  }

  const getInitials = (name: string | null | undefined) => {
    if (!name) return '?'
    return name
      .split(' ')
      .map((n) => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Profile Settings</h1>
        <p className="text-stone-600 dark:text-stone-400">Manage your account information and preferences</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Account Information</CardTitle>
          <CardDescription>
            Your basic account details and authentication information
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center gap-4">
            <Avatar className="h-16 w-16">
              <AvatarImage src={session?.user?.image || undefined} />
              <AvatarFallback className="text-lg">
                {getInitials(session?.user?.name)}
              </AvatarFallback>
            </Avatar>
            <div>
              <h3 className="text-lg font-semibold">{session?.user?.name || 'Unknown'}</h3>
              <p className="text-stone-600 dark:text-stone-400">{session?.user?.email}</p>
            </div>
          </div>

          {!isEditing && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="text-sm font-medium text-stone-700 dark:text-stone-300">Name</label>
                <p className="mt-1 text-sm text-stone-900 dark:text-stone-100">{session?.user?.name || 'Not provided'}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-stone-700 dark:text-stone-300">Email</label>
                <p className="mt-1 text-sm text-stone-900 dark:text-stone-100">{session?.user?.email}</p>
              </div>
            </div>
          )}

          <div className="pt-4 border-t">
            {!isEditing ? (
              <Button onClick={handleEditToggle}>
                Edit Profile
              </Button>
            ) : (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor="name" className="text-sm font-medium">
                      Name
                    </Label>
                    <Input
                      id="name"
                      value={formData.name}
                      onChange={(e) => handleInputChange('name', e.target.value)}
                      placeholder="Enter your name"
                      className={errors.name ? 'border-red-500' : ''}
                    />
                    {errors.name && (
                      <p className="text-xs text-red-600 mt-1">{errors.name}</p>
                    )}
                  </div>
                  <div>
                    <Label htmlFor="email" className="text-sm font-medium">
                      Email
                    </Label>
                    <Input
                      id="email"
                      type="email"
                      value={formData.email}
                      onChange={(e) => handleInputChange('email', e.target.value)}
                      placeholder="Enter your email"
                      className={errors.email ? 'border-red-500' : ''}
                    />
                    {errors.email && (
                      <p className="text-xs text-red-600 mt-1">{errors.email}</p>
                    )}
                  </div>
                </div>
                <div>
                  <Label htmlFor="image" className="text-sm font-medium">
                    Profile Image URL
                  </Label>
                  <Input
                    id="image"
                    type="url"
                    value={formData.image}
                    onChange={(e) => handleInputChange('image', e.target.value)}
                    placeholder="Enter image URL (optional)"
                    className={errors.image ? 'border-red-500' : ''}
                  />
                  {errors.image && (
                    <p className="text-xs text-red-600 mt-1">{errors.image}</p>
                  )}
                  <p className="text-xs text-stone-500 dark:text-stone-400 mt-1">
                    Leave empty to use default avatar with initials
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button onClick={handleSave} disabled={isLoading}>
                    {isLoading ? 'Saving...' : 'Save Changes'}
                  </Button>
                  <Button variant="outline" onClick={handleEditToggle} disabled={isLoading}>
                    Cancel
                  </Button>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Account Status</CardTitle>
          <CardDescription>
            Your account verification and authentication status
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <Badge variant="default" className="bg-green-100 text-green-800">
              Active
            </Badge>
            <span className="text-sm text-stone-600 dark:text-stone-400">
              Account is active and verified
            </span>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}