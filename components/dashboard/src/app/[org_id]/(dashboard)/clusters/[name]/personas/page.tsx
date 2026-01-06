'use client'

import React from 'react'
import { useParams } from 'next/navigation'
import { useOrganization } from '@/components/organization-provider'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ResourceHeader } from '@/components/ui/resource-header'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Users, Plus, MessageCircle, Palette, Clock, MoreHorizontal, Eye, Edit, Trash2, Search } from 'lucide-react'
import Link from 'next/link'
import { usePersonas, useDeletePersona } from '@/hooks/use-personas'
import { useWatchPersonas } from '@/hooks/use-watch'
import { EventsActivity } from '@/components/ui/events-activity'
import { useRouter } from 'next/navigation'

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

export default function ClusterPersonas() {
  const router = useRouter()
  const params = useParams()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const [search, setSearch] = React.useState('')
  const [toneFilter, setToneFilter] = React.useState<string>('all')

  // Fetch all personas with real-time updates
  const { data: personasResponse, isLoading, error } = usePersonas({ clusterName, limit: 100 })
  const deletePersona = useDeletePersona(clusterName)

  // Enable real-time updates via SSE watch
  useWatchPersonas()

  const allPersonas = personasResponse?.data || []
  
  // Filter personas based on search and tone
  const filteredPersonas = allPersonas.filter((persona: any) => {
    const searchQuery = search.toLowerCase()
    const matchesSearch = !search ||
      persona.metadata.name.toLowerCase().includes(searchQuery) ||
      (persona.spec.displayName || '').toLowerCase().includes(searchQuery)

    const matchesTone = toneFilter === 'all' ||
      (persona.spec.tone || '').toLowerCase() === toneFilter.toLowerCase()

    return matchesSearch && matchesTone
  })
  
  // For now, show filtered personas (TODO: implement proper cluster-scoped filtering)
  const clusterPersonas = filteredPersonas
  
  // Get unique tones for filter dropdown
  const tones = React.useMemo(() => {
    const uniqueTones = Array.from(new Set(
      allPersonas.map((persona: any) => persona.spec.tone).filter(Boolean)
    )) as string[]
    return uniqueTones.sort()
  }, [allPersonas])

  const getStatusColor = (phase?: string) => {
    switch (phase) {
      case 'Ready':
      case 'Available':
        return 'bg-green-100 text-green-800'
      case 'Pending':
      case 'Validating':
        return 'bg-yellow-100 text-yellow-800'
      case 'Failed':
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getToneColor = (tone?: string) => {
    switch (tone) {
      case 'professional':
        return 'bg-blue-100 text-blue-800'
      case 'friendly':
        return 'bg-green-100 text-green-800'
      case 'casual':
        return 'bg-orange-100 text-orange-800'
      case 'formal':
        return 'bg-purple-100 text-purple-800'
      case 'empathetic':
        return 'bg-pink-100 text-pink-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const handleDeletePersona = async (persona: any) => {
    if (!persona || !persona.metadata.name) return

    if (confirm(`Are you sure you want to delete persona "${persona.spec.displayName || persona.metadata.name}"?`)) {
      try {
        await deletePersona.mutateAsync(persona.metadata.name)
      } catch (error) {
        console.error('Failed to delete persona:', error)
        alert('Failed to delete persona. Please try again.')
      }
    }
  }

  return (
    <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          icon={Users}
          title="Personas"
          subtitle="Personalities and preferences agents can use to influence their behavior"
          actions={
            <Button asChild>
              <Link href={getOrgUrl(`/clusters/${clusterName}/personas/new`)}>
                <Plus className="h-4 w-4 mr-2" />
                New Persona
              </Link>
            </Button>
          }
        />

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <div className="relative">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search personas..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-9"
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
            </div>
          </CardContent>
        </Card>

        {/* Loading State */}
        {isLoading && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Users className="h-8 w-8 animate-pulse mx-auto mb-4 text-gray-400" />
                <p className="text-gray-600">Loading personas...</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Error State */}
        {error && (
          <Card>
            <CardContent className="flex items-center justify-center py-16">
              <div className="text-center">
                <Users className="h-8 w-8 mx-auto mb-4 text-red-400" />
                <p className="text-red-600 mb-2">Failed to load personas</p>
                <p className="text-gray-600 text-sm">{error.message}</p>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Personas List */}
        {!isLoading && !error && (
          <>
            {clusterPersonas.length === 0 ? (
              /* Empty State */
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-16">
                  <Users className="h-16 w-16 text-gray-400 mb-4" />
                  <CardTitle className="text-xl mb-2">No personas yet</CardTitle>
                  <CardDescription className="text-center max-w-md mb-6">
                    Personas define the behavior, knowledge, and communication style 
                    for AI agents. Create your first persona to get started.
                  </CardDescription>
                  <Button asChild>
                    <Link href={getOrgUrl(`/clusters/${clusterName}/personas/new`)}>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Your First Persona
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardHeader>
                  <CardTitle>Personas ({clusterPersonas.length})</CardTitle>
                  <CardDescription>Language personas in this cluster</CardDescription>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Tone</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Age</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {clusterPersonas.map((persona: any) => (
                        <TableRow key={persona.metadata.name}>
                          <TableCell className="font-medium">
                            <Link
                              href={getOrgUrl(`/clusters/${clusterName}/personas/${persona.metadata.name}`)}
                              className="hover:underline"
                            >
                              {persona.spec.displayName || persona.metadata.name}
                            </Link>
                          </TableCell>
                          <TableCell>
                            {persona.spec.tone ? (
                              <Badge className={getToneColor(persona.spec.tone)} variant="secondary">
                                {persona.spec.tone}
                              </Badge>
                            ) : (
                              <span className="text-sm text-muted-foreground">-</span>
                            )}
                          </TableCell>
                          <TableCell>
                            <Badge className={getStatusColor(persona.status?.phase)}>
                              {persona.status?.phase || 'Unknown'}
                            </Badge>
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
                                  <Link href={getOrgUrl(`/clusters/${clusterName}/personas/${persona.metadata.name}`)}>
                                    <Eye className="h-4 w-4 mr-2" />
                                    View Details
                                  </Link>
                                </DropdownMenuItem>
                                <DropdownMenuItem asChild>
                                  <Link href={getOrgUrl(`/clusters/${clusterName}/personas/${persona.metadata.name}/edit`)}>
                                    <Edit className="h-4 w-4 mr-2" />
                                    Edit
                                  </Link>
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  className="text-destructive"
                                  onClick={() => handleDeletePersona(persona)}
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
                </CardContent>
              </Card>
            )}
          </>
        )}

        {/* Persona Events */}
        <EventsActivity
          title="Persona Events"
          description="Recent events for personas in this cluster"
          clusterName={clusterName}
          resourceType="persona"
          limit={10}
          showNamespace={false}
        />
    </div>
  )
}