'use client'

import { useState } from 'react'
import Link from 'next/link'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { 
  Plus, Search, Users, CheckCircle, AlertCircle, 
  Clock, Activity, MoreHorizontal, Edit, Trash2, Eye,
  User, Briefcase, GraduationCap, Heart, Bot
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { usePersonas, useDeletePersona } from '@/hooks/use-personas'
import { LanguagePersona } from '@/types/persona'
import { Skeleton } from '@/components/ui/skeleton'

function formatTimeAgo(timestamp?: string | Date) {
  if (!timestamp) return 'Unknown'
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (days > 0) return `${days} day${days !== 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours !== 1 ? 's' : ''} ago`
  if (minutes > 0) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`
  return 'Just now'
}

function getToneIcon(tone?: string) {
  if (!tone) return <User className="h-4 w-4 text-gray-500" />
  
  const lowerTone = tone.toLowerCase()
  if (lowerTone.includes('professional') || lowerTone.includes('business')) {
    return <Briefcase className="h-4 w-4 text-blue-500" />
  } else if (lowerTone.includes('friendly') || lowerTone.includes('warm')) {
    return <Heart className="h-4 w-4 text-pink-500" />
  } else if (lowerTone.includes('technical') || lowerTone.includes('expert')) {
    return <GraduationCap className="h-4 w-4 text-purple-500" />
  } else if (lowerTone.includes('creative') || lowerTone.includes('inspiring')) {
    return <Heart className="h-4 w-4 text-orange-500" />
  } else {
    return <User className="h-4 w-4 text-gray-500" />
  }
}

interface PersonaTableProps {
  personas: LanguagePersona[]
  onDelete: (name: string) => void
  isDeleting?: boolean
}

function PersonaTable({ personas, onDelete, isDeleting }: PersonaTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Personas ({personas.length})</CardTitle>
        <CardDescription>Language personas in your organization</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Tone</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Usage</TableHead>
              <TableHead>Age</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {personas.map((persona) => (
              <TableRow key={persona.metadata.name}>
                <TableCell className="font-medium">
                  <div className="flex items-center space-x-2">
                    <Users className="h-4 w-4 text-purple-500" />
                    <Link 
                      href={`/personas/${persona.metadata.name}`}
                      className="hover:underline"
                    >
                      {persona.metadata.name}
                    </Link>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-2">
                    {getToneIcon(persona.spec.tone)}
                    <Badge variant="outline">{persona.spec.tone || 'Not specified'}</Badge>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm max-w-xs truncate">
                    {persona.spec.description || 'No description'}
                  </span>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-2">
                    <CheckCircle className="h-4 w-4 text-green-500" />
                    <Badge variant="default" className="bg-green-100 text-green-800">Active</Badge>
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center space-x-1">
                    <Bot className="h-3 w-3 text-blue-500" />
                    <span className="text-sm">
                      {persona.status?.agentReferences?.length || 0} agents
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted-foreground">
                    {formatTimeAgo(persona.metadata.creationTimestamp)}
                  </span>
                </TableCell>
                <TableCell className="text-right">
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" className="h-8 w-8 p-0">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem asChild>
                        <Link href={`/personas/${persona.metadata.name}`}>
                          <Eye className="h-4 w-4 mr-2" />
                          View Details
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={`/personas/${persona.metadata.name}/edit`}>
                          <Edit className="h-4 w-4 mr-2" />
                          Edit
                        </Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => {
                          if (confirm(`Are you sure you want to delete persona "${persona.metadata.name}"?`)) {
                            onDelete(persona.metadata.name!)
                          }
                        }}
                        disabled={isDeleting}
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {personas.length === 0 && (
          <div className="text-center py-8">
            <Users className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">No personas found</h3>
            <p className="text-muted-foreground mb-4">
              Create your first language persona to get started.
            </p>
            <Link href="/personas/new">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Create Persona
              </Button>
            </Link>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default function PersonasPage() {
  const [search, setSearch] = useState('')
  const [toneFilter, setToneFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'name' | 'tone' | 'usage' | 'age'>('name')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  const { 
    data: personasResponse, 
    isLoading, 
    error,
    refetch 
  } = usePersonas({
    search: search || undefined,
    tone: toneFilter !== 'all' ? [toneFilter] : undefined,
    sortBy,
    sortOrder,
    limit: 100,
  })

  const deletePersona = useDeletePersona()

  const personas = personasResponse?.data || []
  const total = personasResponse?.total || 0

  // Get unique tones for filter dropdown
  const tones = Array.from(new Set(personas.map((persona: LanguagePersona) => persona.spec.tone).filter(Boolean))).sort() as string[]

  // Stats calculations
  const totalAgentUsage = personas.reduce((sum: number, p: LanguagePersona) => sum + (p.status?.agentReferences?.length || 0), 0)
  const personasWithTone = personas.filter((p: LanguagePersona) => p.spec.tone).length
  const personasWithExamples = personas.filter((p: LanguagePersona) => p.spec.examples && p.spec.examples.length > 0).length

  const handleDelete = async (name: string) => {
    try {
      await deletePersona.mutateAsync(name)
      refetch()
    } catch (error) {
      console.error('Failed to delete persona:', error)
      alert('Failed to delete persona. Please try again.')
    }
  }

  if (isLoading) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-9 w-32" />
          </div>
          <div className="grid gap-4 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">Failed to load personas</h3>
            <p className="text-muted-foreground mb-4">
              There was an error loading your language personas.
            </p>
            <Button onClick={() => refetch()}>
              Try Again
            </Button>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Language Personas</h1>
            <p className="text-muted-foreground">
              Define personality traits and communication styles for your agents
            </p>
          </div>
          <Link href="/personas/new">
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Persona
            </Button>
          </Link>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Personas</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{total}</div>
              <p className="text-xs text-muted-foreground">
                Available personas
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Agent Usage</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalAgentUsage}</div>
              <p className="text-xs text-muted-foreground">
                Agents using personas
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">With Tone</CardTitle>
              <Heart className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{personasWithTone}</div>
              <p className="text-xs text-muted-foreground">
                {total > 0 ? Math.round((personasWithTone / total) * 100) : 0}% configured
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">With Examples</CardTitle>
              <GraduationCap className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{personasWithExamples}</div>
              <p className="text-xs text-muted-foreground">
                Have example prompts
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search personas..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-8"
                  />
                </div>
              </div>
              
              <Select value={toneFilter} onValueChange={setToneFilter}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="All Tones" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Tones</SelectItem>
                  {tones.map((tone) => (
                    <SelectItem key={tone} value={tone}>
                      {tone}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={`${sortBy}-${sortOrder}`} onValueChange={(value) => {
                const [newSortBy, newSortOrder] = value.split('-') as [typeof sortBy, typeof sortOrder]
                setSortBy(newSortBy)
                setSortOrder(newSortOrder)
              }}>
                <SelectTrigger className="w-full md:w-48">
                  <SelectValue placeholder="Sort by" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="name-asc">Name (A-Z)</SelectItem>
                  <SelectItem value="name-desc">Name (Z-A)</SelectItem>
                  <SelectItem value="tone-asc">Tone (A-Z)</SelectItem>
                  <SelectItem value="tone-desc">Tone (Z-A)</SelectItem>
                  <SelectItem value="usage-desc">Usage (Most)</SelectItem>
                  <SelectItem value="usage-asc">Usage (Least)</SelectItem>
                  <SelectItem value="age-desc">Newest</SelectItem>
                  <SelectItem value="age-asc">Oldest</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* Personas Table */}
        <PersonaTable 
          personas={personas} 
          onDelete={handleDelete}
          isDeleting={deletePersona.isPending}
        />
      </div>
    </AuthenticatedLayout>
  )
}