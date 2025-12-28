---
name: nextjs-dashboard-patterns
description: Next.js 15 App Router patterns, React components, API routes, and state management for the Language Operator dashboard
---

# Next.js Dashboard Patterns

## Purpose

Provides consistent patterns for building the Language Operator dashboard with Next.js 15 App Router, TypeScript, Tailwind CSS, and shadcn/ui components. Covers component development, API route patterns, state management, and integration with Kubernetes APIs.

## When to Use This Skill

Automatically activates when:
- Working with dashboard components in `components/dashboard/src/`
- Creating or modifying React components and pages
- Implementing API routes and data fetching
- Styling with Tailwind CSS and shadcn/ui
- Working with state management and React hooks

## Quick Start

### New Feature Checklist

- [ ] Create page in `src/app/[feature]/`
- [ ] Implement React Server Components where possible
- [ ] Add API routes in `src/app/api/[endpoint]/`
- [ ] Create reusable components in `src/components/`
- [ ] Use shadcn/ui components from `@/components/ui`
- [ ] Add proper TypeScript types
- [ ] Implement proper error handling
- [ ] Add loading states and skeletons

## Core Principles

### 1. Next.js 15 App Router with Server Components

```typescript
// app/clusters/[name]/agents/page.tsx - Server Component
import { Suspense } from 'react'
import { AgentList } from '@/components/agents/agent-list'
import { AgentListSkeleton } from '@/components/agents/agent-list-skeleton'

interface PageProps {
  params: { name: string }
  searchParams: { search?: string }
}

export default async function AgentsPage({ params, searchParams }: PageProps) {
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Agents</h1>
        <CreateAgentButton clusterName={params.name} />
      </div>
      
      <Suspense fallback={<AgentListSkeleton />}>
        <AgentList 
          clusterName={params.name}
          searchQuery={searchParams.search}
        />
      </Suspense>
    </div>
  )
}
```

### 2. API Routes with Organization Context

```typescript
// app/api/clusters/[name]/agents/route.ts
import { NextRequest } from 'next/server'
import { z } from 'zod'
import { getKubeConfig } from '@/lib/kube-config'
import { withOrganizationContext } from '@/lib/api/with-organization'

const GetAgentsSchema = z.object({
  search: z.string().optional(),
  namespace: z.string().optional(),
})

export async function GET(
  request: NextRequest,
  { params }: { params: { name: string } }
) {
  return withOrganizationContext(async (org) => {
    try {
      const { searchParams } = new URL(request.url)
      const query = GetAgentsSchema.parse({
        search: searchParams.get('search') || undefined,
        namespace: searchParams.get('namespace') || undefined,
      })

      const kubeConfig = await getKubeConfig(org)
      const agents = await kubeConfig.listAgents({
        clusterName: params.name,
        ...query,
      })

      return Response.json({
        success: true,
        data: agents,
      })
    } catch (error) {
      console.error('Error fetching agents:', error)
      return Response.json(
        { success: false, error: 'Failed to fetch agents' },
        { status: 500 }
      )
    }
  })
}
```

### 3. React Query for Data Fetching

```typescript
// hooks/use-agents.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { fetchWithOrganization } from '@/lib/api-client'
import { Agent } from '@/types/agent'

export function useAgents(clusterName: string, searchQuery?: string) {
  return useQuery({
    queryKey: ['agents', clusterName, searchQuery],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (searchQuery) params.append('search', searchQuery)
      
      const response = await fetchWithOrganization(
        `/api/clusters/${clusterName}/agents?${params}`
      )
      if (!response.ok) throw new Error('Failed to fetch agents')
      
      const result = await response.json()
      return result.data as Agent[]
    },
    staleTime: 30 * 1000, // 30 seconds
    refetchOnWindowFocus: false,
  })
}

export function useDeleteAgent(clusterName: string) {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: async (agentName: string) => {
      const response = await fetchWithOrganization(
        `/api/clusters/${clusterName}/agents/${agentName}`,
        { method: 'DELETE' }
      )
      if (!response.ok) throw new Error('Failed to delete agent')
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ 
        queryKey: ['agents', clusterName] 
      })
    },
  })
}
```

### 4. shadcn/ui Component Patterns

```typescript
// components/agents/agent-card.tsx
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { MoreVertical, Play, Edit, Trash } from 'lucide-react'
import { Agent } from '@/types/agent'
import { getStatusColor, getStatusIcon } from './utils'

interface AgentCardProps {
  agent: Agent
  clusterName: string
  onDelete: (name: string) => void
  onExecute: (name: string) => void
}

export function AgentCard({ agent, clusterName, onDelete, onExecute }: AgentCardProps) {
  return (
    <Card className="transition-all hover:shadow-md">
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            {getStatusIcon(agent)}
            <div>
              <h3 className="font-semibold text-lg">{agent.metadata.name}</h3>
              <Badge className={getStatusColor(agent)}>
                {agent.status?.phase || 'Unknown'}
              </Badge>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onExecute(agent.metadata.name)}>
                <Play className="h-4 w-4 mr-2" />
                Execute
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link href={`/clusters/${clusterName}/agents/${agent.metadata.name}/edit`}>
                  <Edit className="h-4 w-4 mr-2" />
                  Edit
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem 
                onClick={() => onDelete(agent.metadata.name)}
                className="text-destructive"
              >
                <Trash className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      
      <CardContent>
        <p className="text-muted-foreground text-sm">
          {agent.spec.description || 'No description provided'}
        </p>
      </CardContent>
      
      <CardFooter className="pt-4 border-t">
        <div className="flex items-center text-xs text-muted-foreground space-x-4">
          <span>Model: {agent.spec.modelName}</span>
          <span>Mode: {agent.spec.executionMode}</span>
        </div>
      </CardFooter>
    </Card>
  )
}
```

## Common Patterns

### Pattern 1: Layout with Navigation

```typescript
// components/layout/authenticated-layout.tsx
import { Sidebar } from './sidebar'
import { Header } from './header'
import { Toaster } from '@/components/ui/toaster'

interface AuthenticatedLayoutProps {
  children: React.ReactNode
}

export function AuthenticatedLayout({ children }: AuthenticatedLayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="flex">
        <Sidebar />
        <main className="flex-1 p-6">
          {children}
        </main>
      </div>
      <Toaster />
    </div>
  )
}
```

### Pattern 2: Form Handling with Zod

```typescript
// components/forms/create-agent-form.tsx
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'

const CreateAgentSchema = z.object({
  name: z.string().min(1, 'Name is required').regex(/^[a-z0-9-]+$/, 'Invalid name format'),
  modelName: z.enum(['claude-3-sonnet', 'claude-3-haiku', 'claude-3-opus']),
  executionMode: z.enum(['scheduled', 'realtime', 'manual']).default('scheduled'),
  schedule: z.string().optional(),
  description: z.string().optional(),
})

type CreateAgentData = z.infer<typeof CreateAgentSchema>

interface CreateAgentFormProps {
  onSubmit: (data: CreateAgentData) => void
  isLoading?: boolean
}

export function CreateAgentForm({ onSubmit, isLoading }: CreateAgentFormProps) {
  const form = useForm<CreateAgentData>({
    resolver: zodResolver(CreateAgentSchema),
    defaultValues: {
      executionMode: 'scheduled',
    },
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="my-agent" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        
        <FormField
          control={form.control}
          name="modelName"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Model</FormLabel>
              <Select onValueChange={field.onChange} defaultValue={field.value}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select a model" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="claude-3-sonnet">Claude 3 Sonnet</SelectItem>
                  <SelectItem value="claude-3-haiku">Claude 3 Haiku</SelectItem>
                  <SelectItem value="claude-3-opus">Claude 3 Opus</SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />

        <Button type="submit" disabled={isLoading}>
          {isLoading ? 'Creating...' : 'Create Agent'}
        </Button>
      </form>
    </Form>
  )
}
```

### Pattern 3: Error Boundary and Loading States

```typescript
// components/ui/error-boundary.tsx
'use client'

import { Component, ReactNode } from 'react'
import { Button } from './button'
import { AlertCircle } from 'lucide-react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback || (
        <div className="flex flex-col items-center justify-center p-8 text-center">
          <AlertCircle className="h-16 w-16 text-destructive mb-4" />
          <h2 className="text-xl font-semibold mb-2">Something went wrong</h2>
          <p className="text-muted-foreground mb-4">
            {this.state.error?.message || 'An unexpected error occurred'}
          </p>
          <Button onClick={() => window.location.reload()}>
            Reload page
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
```

## Resource Files

For detailed information, see:
- [API Patterns](resources/api-patterns.md) - REST API design and error handling
- [Component Library](resources/component-patterns.md) - Reusable component patterns
- [State Management](resources/state-patterns.md) - React Query and context patterns

## Anti-Patterns to Avoid

❌ **Client Components for static content** - Use Server Components when possible
❌ **Missing error boundaries** - Always wrap components that might error
❌ **Hardcoded API URLs** - Use environment variables and api-client
❌ **Missing loading states** - Always show loading feedback
❌ **Inline styles** - Use Tailwind utility classes
❌ **Missing TypeScript types** - Type all props and API responses
❌ **Direct fetch calls** - Use fetchWithOrganization wrapper
❌ **Missing input validation** - Always validate with Zod schemas

## Quick Reference

| Need to... | Use this |
|-----------|----------|
| Create new page | Add to `src/app/[route]/page.tsx` |
| Add API endpoint | Create `src/app/api/[endpoint]/route.ts` |
| Fetch data | Use React Query hooks in `hooks/` |
| Create component | Use shadcn/ui + TypeScript in `components/` |
| Style component | Use Tailwind utility classes |
| Validate forms | Use Zod + react-hook-form |
| Handle errors | Use ErrorBoundary and try/catch |
| Show loading | Use Suspense and skeleton components |

## Integration with Language Operator

This skill integrates with:
- **Kubernetes APIs**: All data fetched from Kubernetes API via organization context
- **Real-time updates**: WebSocket connections for live cluster status
- **Authentication**: Organization-based access control and RBAC
- **Telemetry**: ClickHouse integration for metrics and trace visualization