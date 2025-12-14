'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { 
  ArrowLeft, Bot, Save, Eye, Plus, X, 
  Settings, Zap, Network, Shield 
} from 'lucide-react'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { useCreateAgent } from '@/hooks/use-agents'
import { LanguageAgentFormData, LanguageAgent } from '@/types/agent'
import { Skeleton } from '@/components/ui/skeleton'
import { useToast } from '@/hooks/use-toast'

// Form validation schema
const agentFormSchema = z.object({
  name: z.string()
    .min(1, 'Name is required')
    .max(253, 'Name must be 253 characters or less')
    .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, 'Name must be a valid Kubernetes resource name'),
  executionMode: z.enum(['autonomous', 'interactive', 'scheduled', 'event-driven']),
  replicas: z.number().min(1, 'At least 1 replica is required').max(10, 'Maximum 10 replicas allowed'),
  
  // Model configuration
  modelName: z.string().min(1, 'Model name is required'),
  modelProvider: z.string().optional(),
  modelEndpoint: z.string().optional(),
  
  // Persona configuration (optional)
  personaName: z.string().optional(),
  personaTone: z.string().optional(),
  personaInstructions: z.string().optional(),
  
  // Tools (simplified for now)
  selectedTools: z.array(z.string()),
  
  // Resources
  cpuRequest: z.string().optional(),
  memoryRequest: z.string().optional(),
  cpuLimit: z.string().optional(),
  memoryLimit: z.string().optional(),
  
  // Scaling
  minReplicas: z.number().min(1).optional(),
  maxReplicas: z.number().min(1).optional(),
  targetCPUUtilization: z.number().min(1).max(100).optional(),
  
  // Networking
  enableIngress: z.boolean(),
  ingressHost: z.string().optional(),
  ingressPath: z.string().optional(),
  enableTLS: z.boolean(),
})

type AgentFormValues = z.infer<typeof agentFormSchema>

// Available tools (in a real app, this would come from an API)
const availableTools = [
  'web-search',
  'calculator',
  'file-reader',
  'image-generator',
  'code-interpreter',
  'database-query',
]

export default function CreateAgentPage() {
  const router = useRouter()
  const [activeTab, setActiveTab] = useState('basic')
  const [showYamlPreview, setShowYamlPreview] = useState(false)
  const { toast } = useToast()
  
  const createAgent = useCreateAgent()
  
  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentFormSchema),
    defaultValues: {
      name: '',
      executionMode: 'autonomous',
      replicas: 1,
      modelName: '',
      modelProvider: '',
      modelEndpoint: '',
      personaName: '',
      personaTone: '',
      personaInstructions: '',
      selectedTools: [],
      cpuRequest: '100m',
      memoryRequest: '128Mi',
      cpuLimit: '500m',
      memoryLimit: '512Mi',
      minReplicas: 1,
      maxReplicas: 3,
      targetCPUUtilization: 70,
      enableIngress: false,
      ingressHost: '',
      ingressPath: '/',
      enableTLS: false,
    },
  })

  const watchedValues = form.watch()

  // Generate YAML preview
  const generateYamlPreview = (values: AgentFormValues): string => {
    const agent: Partial<LanguageAgent> = {
      apiVersion: 'langop.io/v1alpha1',
      kind: 'LanguageAgent',
      metadata: {
        name: values.name || 'example-agent',
        namespace: 'default', // This will be set by the API based on user's org
      },
      spec: {
        executionMode: values.executionMode,
        replicas: values.replicas,
        model: {
          name: values.modelName,
          ...(values.modelProvider && { provider: values.modelProvider }),
          ...(values.modelEndpoint && { endpoint: values.modelEndpoint }),
        },
        ...(values.personaName && {
          persona: {
            name: values.personaName,
            ...(values.personaTone && { tone: values.personaTone }),
            ...(values.personaInstructions && { instructions: values.personaInstructions }),
          },
        }),
        ...(values.selectedTools.length > 0 && {
          tools: values.selectedTools.map(name => ({ name })),
        }),
        ...(values.cpuRequest || values.memoryRequest || values.cpuLimit || values.memoryLimit) && {
          resources: {
            ...(values.cpuRequest || values.memoryRequest) && {
              requests: {
                ...(values.cpuRequest && { cpu: values.cpuRequest }),
                ...(values.memoryRequest && { memory: values.memoryRequest }),
              },
            },
            ...(values.cpuLimit || values.memoryLimit) && {
              limits: {
                ...(values.cpuLimit && { cpu: values.cpuLimit }),
                ...(values.memoryLimit && { memory: values.memoryLimit }),
              },
            },
          },
        },
        ...(values.minReplicas || values.maxReplicas || values.targetCPUUtilization) && {
          scaling: {
            ...(values.minReplicas && { minReplicas: values.minReplicas }),
            ...(values.maxReplicas && { maxReplicas: values.maxReplicas }),
            ...(values.targetCPUUtilization && { targetCPUUtilization: values.targetCPUUtilization }),
          },
        },
        ...(values.enableIngress) && {
          networking: {
            ingress: {
              enabled: true,
              ...(values.ingressHost && { host: values.ingressHost }),
              ...(values.ingressPath && { path: values.ingressPath }),
              tls: values.enableTLS,
            },
          },
        },
      },
    }
    
    return JSON.stringify(agent, null, 2)
  }

  const onSubmit = async (values: AgentFormValues) => {
    try {
      const formData: LanguageAgentFormData = {
        name: values.name,
        namespace: '', // Will be set by API
        executionMode: values.executionMode,
        replicas: values.replicas,
        modelName: values.modelName,
        modelProvider: values.modelProvider,
        modelEndpoint: values.modelEndpoint,
        personaName: values.personaName,
        personaTone: values.personaTone,
        personaInstructions: values.personaInstructions,
        selectedTools: values.selectedTools,
        cpuRequest: values.cpuRequest,
        memoryRequest: values.memoryRequest,
        cpuLimit: values.cpuLimit,
        memoryLimit: values.memoryLimit,
        minReplicas: values.minReplicas,
        maxReplicas: values.maxReplicas,
        targetCPUUtilization: values.targetCPUUtilization,
        enableIngress: values.enableIngress,
        ingressHost: values.ingressHost,
        ingressPath: values.ingressPath,
        enableTLS: values.enableTLS,
      }

      await createAgent.mutateAsync(formData)
      
      toast({
        title: 'Agent created successfully',
        description: `Agent "${values.name}" has been created and is starting up.`,
      })
      
      router.push(`/agents/${values.name}`)
    } catch (error) {
      console.error('Failed to create agent:', error)
      toast({
        title: 'Failed to create agent',
        description: error instanceof Error ? error.message : 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }

  const addTool = (toolName: string) => {
    const currentTools = form.getValues('selectedTools')
    if (!currentTools.includes(toolName)) {
      form.setValue('selectedTools', [...currentTools, toolName])
    }
  }

  const removeTool = (toolName: string) => {
    const currentTools = form.getValues('selectedTools')
    form.setValue('selectedTools', currentTools.filter(t => t !== toolName))
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Link href="/agents">
              <Button variant="outline" size="icon">
                <ArrowLeft className="h-4 w-4" />
              </Button>
            </Link>
            <Bot className="h-8 w-8 text-blue-500" />
            <div>
              <h1 className="text-3xl font-bold">Create Language Agent</h1>
              <p className="text-muted-foreground">
                Configure and deploy a new language agent
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowYamlPreview(!showYamlPreview)}
            >
              <Eye className="h-4 w-4 mr-2" />
              {showYamlPreview ? 'Hide' : 'Show'} YAML
            </Button>
          </div>
        </div>

        <div className="grid gap-6 lg:grid-cols-3">
          {/* Main Form */}
          <div className={showYamlPreview ? "lg:col-span-2" : "lg:col-span-3"}>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                <Tabs value={activeTab} onValueChange={setActiveTab}>
                  <TabsList className="grid w-full grid-cols-4">
                    <TabsTrigger value="basic">
                      <Settings className="h-4 w-4 mr-2" />
                      Basic
                    </TabsTrigger>
                    <TabsTrigger value="model">
                      <Bot className="h-4 w-4 mr-2" />
                      Model & Tools
                    </TabsTrigger>
                    <TabsTrigger value="resources">
                      <Zap className="h-4 w-4 mr-2" />
                      Resources
                    </TabsTrigger>
                    <TabsTrigger value="networking">
                      <Network className="h-4 w-4 mr-2" />
                      Network
                    </TabsTrigger>
                  </TabsList>

                  {/* Basic Configuration */}
                  <TabsContent value="basic" className="space-y-6">
                    <Card>
                      <CardHeader>
                        <CardTitle>Basic Configuration</CardTitle>
                        <CardDescription>
                          Set up the basic parameters for your language agent
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <FormField
                          control={form.control}
                          name="name"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Agent Name</FormLabel>
                              <FormControl>
                                <Input placeholder="my-agent" {...field} />
                              </FormControl>
                              <FormDescription>
                                Must be a valid Kubernetes resource name (lowercase, numbers, hyphens)
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        <FormField
                          control={form.control}
                          name="executionMode"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Execution Mode</FormLabel>
                              <Select onValueChange={field.onChange} defaultValue={field.value}>
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue placeholder="Select execution mode" />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="worker">Worker</SelectItem>
                                  <SelectItem value="server">Server</SelectItem>
                                  <SelectItem value="hybrid">Hybrid</SelectItem>
                                </SelectContent>
                              </Select>
                              <FormDescription>
                                Worker: Process tasks from queue. Server: Handle HTTP requests. Hybrid: Both.
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        <FormField
                          control={form.control}
                          name="replicas"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Replicas</FormLabel>
                              <FormControl>
                                <Input 
                                  type="number" 
                                  min="1" 
                                  max="10" 
                                  {...field}
                                  onChange={(e) => field.onChange(parseInt(e.target.value))}
                                />
                              </FormControl>
                              <FormDescription>
                                Number of agent instances to run (1-10)
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Model & Tools Configuration */}
                  <TabsContent value="model" className="space-y-6">
                    <Card>
                      <CardHeader>
                        <CardTitle>Model Configuration</CardTitle>
                        <CardDescription>
                          Configure the language model for your agent
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <FormField
                          control={form.control}
                          name="modelName"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Model Name</FormLabel>
                              <FormControl>
                                <Input placeholder="gpt-4" {...field} />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        <div className="grid grid-cols-2 gap-4">
                          <FormField
                            control={form.control}
                            name="modelProvider"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Provider (Optional)</FormLabel>
                                <FormControl>
                                  <Input placeholder="openai" {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="modelEndpoint"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Endpoint (Optional)</FormLabel>
                                <FormControl>
                                  <Input placeholder="https://api.openai.com/v1" {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle>Persona Configuration</CardTitle>
                        <CardDescription>
                          Optional personality and behavior settings
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <FormField
                            control={form.control}
                            name="personaName"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Persona Name (Optional)</FormLabel>
                                <FormControl>
                                  <Input placeholder="helpful-assistant" {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="personaTone"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Tone (Optional)</FormLabel>
                                <FormControl>
                                  <Input placeholder="helpful, professional" {...field} />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>

                        <FormField
                          control={form.control}
                          name="personaInstructions"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Instructions (Optional)</FormLabel>
                              <FormControl>
                                <Textarea 
                                  placeholder="You are a helpful assistant that..."
                                  className="min-h-[100px]"
                                  {...field} 
                                />
                              </FormControl>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle>Tools Configuration</CardTitle>
                        <CardDescription>
                          Select tools and capabilities for your agent
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-4">
                          <div>
                            <Label>Available Tools</Label>
                            <div className="grid grid-cols-2 gap-2 mt-2">
                              {availableTools.map((tool) => (
                                <div
                                  key={tool}
                                  className="flex items-center justify-between p-2 border rounded-lg"
                                >
                                  <span className="text-sm">{tool}</span>
                                  <Button
                                    type="button"
                                    size="sm"
                                    variant={watchedValues.selectedTools.includes(tool) ? "default" : "outline"}
                                    onClick={() => {
                                      if (watchedValues.selectedTools.includes(tool)) {
                                        removeTool(tool)
                                      } else {
                                        addTool(tool)
                                      }
                                    }}
                                  >
                                    {watchedValues.selectedTools.includes(tool) ? (
                                      <X className="h-3 w-3" />
                                    ) : (
                                      <Plus className="h-3 w-3" />
                                    )}
                                  </Button>
                                </div>
                              ))}
                            </div>
                          </div>

                          {watchedValues.selectedTools.length > 0 && (
                            <div>
                              <Label>Selected Tools</Label>
                              <div className="flex flex-wrap gap-2 mt-2">
                                {watchedValues.selectedTools.map((tool) => (
                                  <Badge key={tool} variant="default" className="text-xs">
                                    {tool}
                                    <button
                                      type="button"
                                      onClick={() => removeTool(tool)}
                                      className="ml-2 text-xs"
                                    >
                                      ×
                                    </button>
                                  </Badge>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Resources Configuration */}
                  <TabsContent value="resources" className="space-y-6">
                    <Card>
                      <CardHeader>
                        <CardTitle>Resource Limits</CardTitle>
                        <CardDescription>
                          Configure CPU and memory resources
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-2 gap-4">
                          <FormField
                            control={form.control}
                            name="cpuRequest"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>CPU Request</FormLabel>
                                <FormControl>
                                  <Input placeholder="100m" {...field} />
                                </FormControl>
                                <FormDescription>e.g., 100m, 0.5, 1</FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="cpuLimit"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>CPU Limit</FormLabel>
                                <FormControl>
                                  <Input placeholder="500m" {...field} />
                                </FormControl>
                                <FormDescription>e.g., 500m, 1, 2</FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="memoryRequest"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Memory Request</FormLabel>
                                <FormControl>
                                  <Input placeholder="128Mi" {...field} />
                                </FormControl>
                                <FormDescription>e.g., 128Mi, 1Gi</FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="memoryLimit"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Memory Limit</FormLabel>
                                <FormControl>
                                  <Input placeholder="512Mi" {...field} />
                                </FormControl>
                                <FormDescription>e.g., 512Mi, 2Gi</FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle>Auto Scaling</CardTitle>
                        <CardDescription>
                          Configure automatic scaling behavior
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="grid grid-cols-3 gap-4">
                          <FormField
                            control={form.control}
                            name="minReplicas"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Min Replicas</FormLabel>
                                <FormControl>
                                  <Input 
                                    type="number" 
                                    min="1"
                                    {...field}
                                    onChange={(e) => field.onChange(parseInt(e.target.value))}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="maxReplicas"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Max Replicas</FormLabel>
                                <FormControl>
                                  <Input 
                                    type="number" 
                                    min="1"
                                    {...field}
                                    onChange={(e) => field.onChange(parseInt(e.target.value))}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <FormField
                            control={form.control}
                            name="targetCPUUtilization"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Target CPU %</FormLabel>
                                <FormControl>
                                  <Input 
                                    type="number" 
                                    min="1" 
                                    max="100"
                                    {...field}
                                    onChange={(e) => field.onChange(parseInt(e.target.value))}
                                  />
                                </FormControl>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </CardContent>
                    </Card>
                  </TabsContent>

                  {/* Networking Configuration */}
                  <TabsContent value="networking" className="space-y-6">
                    <Card>
                      <CardHeader>
                        <CardTitle>Ingress Configuration</CardTitle>
                        <CardDescription>
                          Configure external access to your agent
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <FormField
                          control={form.control}
                          name="enableIngress"
                          render={({ field }) => (
                            <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                              <div className="space-y-0.5">
                                <FormLabel className="text-base">
                                  Enable Ingress
                                </FormLabel>
                                <FormDescription>
                                  Expose your agent via HTTP/HTTPS
                                </FormDescription>
                              </div>
                              <FormControl>
                                <Checkbox
                                  checked={field.value}
                                  onCheckedChange={field.onChange}
                                />
                              </FormControl>
                            </FormItem>
                          )}
                        />

                        {watchedValues.enableIngress && (
                          <>
                            <div className="grid grid-cols-2 gap-4">
                              <FormField
                                control={form.control}
                                name="ingressHost"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Host</FormLabel>
                                    <FormControl>
                                      <Input placeholder="my-agent.example.com" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />

                              <FormField
                                control={form.control}
                                name="ingressPath"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Path</FormLabel>
                                    <FormControl>
                                      <Input placeholder="/" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            </div>

                            <FormField
                              control={form.control}
                              name="enableTLS"
                              render={({ field }) => (
                                <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                                  <div className="space-y-0.5">
                                    <FormLabel className="text-base">
                                      Enable TLS
                                    </FormLabel>
                                    <FormDescription>
                                      Use HTTPS with automatic certificate
                                    </FormDescription>
                                  </div>
                                  <FormControl>
                                    <Checkbox
                                      checked={field.value}
                                      onCheckedChange={field.onChange}
                                    />
                                  </FormControl>
                                </FormItem>
                              )}
                            />
                          </>
                        )}
                      </CardContent>
                    </Card>
                  </TabsContent>
                </Tabs>

                {/* Submit Buttons */}
                <div className="flex items-center justify-between">
                  <Link href="/agents">
                    <Button type="button" variant="outline">
                      Cancel
                    </Button>
                  </Link>
                  
                  <Button 
                    type="submit" 
                    disabled={createAgent.isPending}
                    className="ml-auto"
                  >
                    {createAgent.isPending ? (
                      <>Creating...</>
                    ) : (
                      <>
                        <Save className="h-4 w-4 mr-2" />
                        Create Agent
                      </>
                    )}
                  </Button>
                </div>
              </form>
            </Form>
          </div>

          {/* YAML Preview */}
          {showYamlPreview && (
            <div className="lg:col-span-1">
              <Card className="sticky top-6">
                <CardHeader>
                  <CardTitle className="text-sm">YAML Preview</CardTitle>
                  <CardDescription>
                    Generated Kubernetes resource definition
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <pre className="text-xs bg-muted p-4 rounded-lg overflow-auto max-h-[600px]">
                    {generateYamlPreview(watchedValues)}
                  </pre>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}