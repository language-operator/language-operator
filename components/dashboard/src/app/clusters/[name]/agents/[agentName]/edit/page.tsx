'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { 
  ArrowLeft, Bot, Save, Plus, X, Trash2,
  Settings, Zap, Network
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
import { useAgent, useUpdateAgent } from '@/hooks/use-agents'
import { useModels } from '@/hooks/use-models'
import { useTools } from '@/hooks/use-tools'
import { usePersonas } from '@/hooks/use-personas'
import { LanguageAgentFormData, LanguageAgent } from '@/types/agent'
import { useToast } from '@/hooks/use-toast'
import { kubernetesNameValidation } from '@/lib/validation'

// Form validation schema
const agentFormSchema = z.object({
  // Basic fields (matching create form)
  instructions: z.string()
    .min(1, 'Goal is required')
    .min(10, 'Goal must be at least 10 characters')
    .max(5000, 'Goal must be less than 5000 characters'),
  name: kubernetesNameValidation,
  selectedModels: z.array(z.string()).min(1, 'At least one model must be selected'),
  selectedTools: z.array(z.string()),
  selectedPersona: z.string().optional(),
  
  // Resources
  cpuRequest: z.string().optional(),
  memoryRequest: z.string().optional(),
  cpuLimit: z.string().optional(),
  memoryLimit: z.string().optional(),
  
  
  // Network Policies
  egressRules: z.array(z.object({
    description: z.string().optional(),
    dns: z.array(z.string()).optional(),
    cidr: z.string().optional(),
    ports: z.array(z.object({
      port: z.number().min(1).max(65535),
      protocol: z.enum(['TCP', 'UDP'])
    })).optional()
  })).optional(),
})

type AgentFormValues = z.infer<typeof agentFormSchema>


export default function EditClusterAgentPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  const agentName = params?.agentName as string
  
  const [activeTab, setActiveTab] = useState('basic')
  const { toast } = useToast()
  
  const { data: agentResponse, isLoading: isLoadingAgent } = useAgent(agentName, clusterName)
  const updateAgent = useUpdateAgent(clusterName)
  const agent = agentResponse?.data

  // Fetch available data for dropdowns
  const { data: modelsResponse, isLoading: isLoadingModels } = useModels({ clusterName })
  const { data: toolsResponse, isLoading: isLoadingTools } = useTools({ clusterName })
  const { data: personasResponse, isLoading: isLoadingPersonas } = usePersonas({ clusterName })
  
  // Extract data from API responses
  const availableModels = modelsResponse?.data || []
  const availableTools = toolsResponse?.data || []
  const availablePersonas = personasResponse?.data || []
  
  const form = useForm<AgentFormValues>({
    resolver: zodResolver(agentFormSchema),
    defaultValues: {
      instructions: '',
      name: '',
      selectedModels: [],
      selectedTools: [],
      selectedPersona: 'none',
      cpuRequest: '100m',
      memoryRequest: '128Mi',
      cpuLimit: '500m',
      memoryLimit: '512Mi',
      egressRules: [],
    },
  })

  // Populate form when agent data is loaded
  useEffect(() => {
    if (agent) {
      form.reset({
        instructions: agent.spec.instructions || '',
        name: agent.metadata.name,
        selectedModels: agent.spec.modelRefs?.map((m: any) => m.name) || [],
        selectedTools: agent.spec.toolRefs?.map((t: any) => t.name) || [],
        selectedPersona: agent.spec.personaRefs?.[0]?.name || 'none',
        cpuRequest: agent.spec.resources?.requests?.cpu || '100m',
        memoryRequest: agent.spec.resources?.requests?.memory || '128Mi',
        cpuLimit: agent.spec.resources?.limits?.cpu || '500m',
        memoryLimit: agent.spec.resources?.limits?.memory || '512Mi',
        egressRules: agent.spec.egress?.map((rule: any) => ({
          description: rule.description || '',
          dns: rule.to?.dns || [],
          cidr: rule.to?.cidr || '',
          ports: rule.ports || []
        })) || [],
      })
    }
  }, [agent, form])

  const watchedValues = form.watch()

  const onSubmit = async (values: AgentFormValues) => {
    try {
      const formData: LanguageAgentFormData = {
        instructions: values.instructions,
        name: values.name,
        namespace: agent?.metadata.namespace || '',
        selectedModels: values.selectedModels,
        selectedTools: values.selectedTools,
        selectedPersona: values.selectedPersona === 'none' ? undefined : values.selectedPersona,
        cpuRequest: values.cpuRequest,
        memoryRequest: values.memoryRequest,
        cpuLimit: values.cpuLimit,
        memoryLimit: values.memoryLimit,
        egressRules: values.egressRules,
      }

      await updateAgent.mutateAsync({ name: agentName, agent: formData as any })
      
      toast({
        title: 'Agent updated successfully',
        description: `Agent "${values.name}" has been updated.`,
      })
      
      // Redirect to agent detail page
      router.push(`/clusters/${clusterName}/agents/${agentName}`)
    } catch (error) {
      console.error('Failed to update agent:', error)
      toast({
        title: 'Failed to update agent',
        description: error instanceof Error ? error.message : 'An unexpected error occurred.',
        variant: 'destructive',
      })
    }
  }


  const handleCancel = () => {
    router.push(`/clusters/${clusterName}/agents/${agentName}`)
  }

  if (isLoadingAgent) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleCancel}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div className="space-y-2">
              <div className="h-8 w-48 bg-gray-200 rounded animate-pulse"></div>
              <div className="h-4 w-64 bg-gray-200 rounded animate-pulse"></div>
            </div>
          </div>
          <div className="h-96 bg-gray-200 rounded animate-pulse"></div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!agent) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleCancel}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold">Agent Not Found</h1>
              <p className="text-muted-foreground mt-1">
                The agent "{agentName}" was not found in cluster "{clusterName}"
              </p>
            </div>
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
          <div className="flex items-center space-x-4">
            <Button variant="outline" size="icon" onClick={handleCancel}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <Bot className="h-8 w-8 text-blue-500" />
            <div>
              <h1 className="text-3xl font-bold">Edit Language Agent</h1>
              <p className="text-muted-foreground">
                Edit "{agentName}" in the {clusterName} cluster
              </p>
            </div>
          </div>
        </div>

        <div className="grid gap-6">
          {/* Main Form */}
          <div>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                <Tabs value={activeTab} onValueChange={setActiveTab}>
                  <TabsList className="grid w-full grid-cols-3">
                    <TabsTrigger value="basic">
                      <Settings className="h-4 w-4 mr-2" />
                      Basic
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
                        <CardTitle>Agent Configuration</CardTitle>
                        <CardDescription>
                          Define your language agent with instructions, models, tools, and persona
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-6">
                        {/* Instructions - Primary field */}
                        <FormField
                          control={form.control}
                          name="instructions"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Goal *</FormLabel>
                              <FormControl>
                                <Textarea 
                                  placeholder="Enter the goal for your agent (e.g., 'Write a short story', 'Analyze customer feedback', 'Generate test cases')..."
                                  className="min-h-[120px] text-base"
                                  {...field}
                                />
                              </FormControl>
                              <FormDescription>
                                The specific goal or task you want this agent to accomplish
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        {/* Agent Name */}
                        <FormField
                          control={form.control}
                          name="name"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Agent Name *</FormLabel>
                              <FormControl>
                                <Input {...field} disabled />
                              </FormControl>
                              <FormDescription>
                                Name cannot be changed after creation
                              </FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        {/* Models Multi-select */}
                        <FormField
                          control={form.control}
                          name="selectedModels"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Models *</FormLabel>
                              <FormDescription>
                                Select one or more models for your agent
                              </FormDescription>
                              {isLoadingModels ? (
                                <div className="text-sm text-muted-foreground">Loading available models...</div>
                              ) : (
                                <div className="grid grid-cols-1 gap-2 mt-2">
                                  {availableModels.map((model: any) => (
                                    <div key={model.metadata.name} className="flex items-center space-x-2">
                                      <Checkbox
                                        checked={field.value.includes(model.metadata.name)}
                                        onCheckedChange={(checked) => {
                                          if (checked) {
                                            field.onChange([...field.value, model.metadata.name])
                                          } else {
                                            field.onChange(field.value.filter(name => name !== model.metadata.name))
                                          }
                                        }}
                                      />
                                      <div className="flex-1">
                                        <div className="font-medium">{model.metadata.name}</div>
                                        <div className="text-sm text-muted-foreground">{model.spec.provider} - {model.spec.modelName}</div>
                                      </div>
                                    </div>
                                  ))}
                                  {availableModels.length === 0 && (
                                    <div className="text-sm text-muted-foreground">No models available in this cluster</div>
                                  )}
                                </div>
                              )}
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        {/* Tools Multi-select */}
                        <FormField
                          control={form.control}
                          name="selectedTools"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Tools</FormLabel>
                              <FormDescription>
                                Select tools and capabilities for your agent
                              </FormDescription>
                              {isLoadingTools ? (
                                <div className="text-sm text-muted-foreground">Loading available tools...</div>
                              ) : (
                                <div className="grid grid-cols-2 gap-2 mt-2">
                                  {availableTools.map((tool: any) => (
                                    <div key={tool.metadata.name} className="flex items-center space-x-2">
                                      <Checkbox
                                        checked={field.value.includes(tool.metadata.name)}
                                        onCheckedChange={(checked) => {
                                          if (checked) {
                                            field.onChange([...field.value, tool.metadata.name])
                                          } else {
                                            field.onChange(field.value.filter(name => name !== tool.metadata.name))
                                          }
                                        }}
                                      />
                                      <div className="flex-1">
                                        <div className="font-medium">{tool.metadata.name}</div>
                                      </div>
                                    </div>
                                  ))}
                                  {availableTools.length === 0 && (
                                    <div className="text-sm text-muted-foreground">No tools available in this cluster</div>
                                  )}
                                </div>
                              )}
                              <FormMessage />
                            </FormItem>
                          )}
                        />

                        {/* Persona Single Select */}
                        <FormField
                          control={form.control}
                          name="selectedPersona"
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Persona</FormLabel>
                              <Select onValueChange={field.onChange} value={field.value}>
                                <FormControl>
                                  <SelectTrigger>
                                    <SelectValue placeholder="None" />
                                  </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                  <SelectItem value="none">None</SelectItem>
                                  {isLoadingPersonas ? (
                                    <SelectItem value="loading" disabled>Loading personas...</SelectItem>
                                  ) : (
                                    availablePersonas.map((persona: any) => (
                                      <SelectItem key={persona.metadata.name} value={persona.metadata.name}>
                                        {persona.metadata.name}
                                      </SelectItem>
                                    ))
                                  )}
                                </SelectContent>
                              </Select>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
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

                  </TabsContent>

                  {/* Network Policy Configuration */}
                  <TabsContent value="networking" className="space-y-6">
                    <Card>
                      <CardHeader>
                        <CardTitle>Egress Network Policy</CardTitle>
                        <CardDescription>
                          Control external network access for security. By default, agents can access cluster resources but no external endpoints.
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <FormLabel>Egress Rules</FormLabel>
                            <FormDescription>
                              Define which external endpoints this agent can access
                            </FormDescription>
                          </div>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => {
                              const currentRules = form.getValues('egressRules') || []
                              form.setValue('egressRules', [...currentRules, {
                                description: '',
                                dns: [],
                                cidr: '',
                                ports: []
                              }])
                            }}
                          >
                            <Plus className="h-4 w-4 mr-2" />
                            Add Rule
                          </Button>
                        </div>

                        {watchedValues.egressRules && watchedValues.egressRules.length > 0 ? (
                          <div className="space-y-4">
                            {watchedValues.egressRules.map((rule, index) => (
                              <Card key={index} className="p-4">
                                <div className="flex items-start justify-between mb-4">
                                  <div className="flex-1">
                                    <FormField
                                      control={form.control}
                                      name={`egressRules.${index}.description`}
                                      render={({ field }) => (
                                        <FormItem>
                                          <FormLabel>Rule Description</FormLabel>
                                          <FormControl>
                                            <Input 
                                              placeholder="e.g., Allow access to OpenAI API" 
                                              {...field} 
                                            />
                                          </FormControl>
                                          <FormMessage />
                                        </FormItem>
                                      )}
                                    />
                                  </div>
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="ml-2"
                                    onClick={() => {
                                      const currentRules = form.getValues('egressRules') || []
                                      const newRules = currentRules.filter((_, i) => i !== index)
                                      form.setValue('egressRules', newRules)
                                    }}
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </div>

                                <div className="grid grid-cols-2 gap-4">
                                  <FormField
                                    control={form.control}
                                    name={`egressRules.${index}.dns`}
                                    render={({ field }) => (
                                      <FormItem>
                                        <FormLabel>DNS Names</FormLabel>
                                        <FormControl>
                                          <Textarea
                                            placeholder="api.openai.com&#10;*.googleapis.com&#10;api.anthropic.com"
                                            className="min-h-[80px]"
                                            value={Array.isArray(field.value) ? field.value.join('\n') : ''}
                                            onChange={(e) => {
                                              const lines = e.target.value.split('\n').filter(line => line.trim())
                                              field.onChange(lines)
                                            }}
                                          />
                                        </FormControl>
                                        <FormDescription>
                                          One domain per line. Supports wildcards with *
                                        </FormDescription>
                                        <FormMessage />
                                      </FormItem>
                                    )}
                                  />

                                  <FormField
                                    control={form.control}
                                    name={`egressRules.${index}.cidr`}
                                    render={({ field }) => (
                                      <FormItem>
                                        <FormLabel>CIDR Block</FormLabel>
                                        <FormControl>
                                          <Input 
                                            placeholder="e.g., 10.0.0.0/8 or 192.168.1.0/24" 
                                            {...field} 
                                          />
                                        </FormControl>
                                        <FormDescription>
                                          IP address range in CIDR notation
                                        </FormDescription>
                                        <FormMessage />
                                      </FormItem>
                                    )}
                                  />
                                </div>
                              </Card>
                            ))}
                          </div>
                        ) : (
                          <div className="text-center py-8 text-muted-foreground">
                            <Network className="h-8 w-8 mx-auto mb-2 opacity-50" />
                            <p>No egress rules configured</p>
                            <p className="text-sm">Agent will only be able to access cluster resources</p>
                          </div>
                        )}

                        {watchedValues.egressRules && watchedValues.egressRules.length > 0 && (
                          <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-950/20 rounded-lg">
                            <h4 className="text-sm font-medium mb-2">Common Egress Rules</h4>
                            <div className="grid grid-cols-1 gap-2 text-sm">
                              <div>
                                <strong>AI APIs:</strong> api.openai.com, api.anthropic.com, *.googleapis.com
                              </div>
                              <div>
                                <strong>Private Networks:</strong> 10.0.0.0/8, 192.168.0.0/16, 172.16.0.0/12
                              </div>
                              <div>
                                <strong>Internet Access:</strong> 0.0.0.0/0 (use with caution)
                              </div>
                            </div>
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  </TabsContent>
                </Tabs>

                {/* Submit Buttons */}
                <div className="flex items-center justify-between">
                  <Button type="button" variant="outline" onClick={handleCancel}>
                    Cancel
                  </Button>
                  
                  <Button 
                    type="submit" 
                    disabled={updateAgent.isPending}
                    className="ml-auto"
                  >
                    {updateAgent.isPending ? (
                      <>Updating...</>
                    ) : (
                      <>
                        <Save className="h-4 w-4 mr-2" />
                        Update Agent
                      </>
                    )}
                  </Button>
                </div>
              </form>
            </Form>
          </div>

        </div>
      </div>
    </AuthenticatedLayout>
  )
}