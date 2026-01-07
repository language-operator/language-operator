'use client'

import { useState, useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { 
  Save, Settings, Zap, Network
} from 'lucide-react'
import { ResourceNetworkPolicyForm } from '@/components/forms/resource-network-policy-form'
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
import { useOrganization } from '@/components/organization-provider'

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
  
  // Ingress rules - agents accept connections from external/gateway
  ingressRules: z.array(z.object({
    description: z.string().optional(),
    from: z.enum(['agents', 'tools', 'models', 'cluster', 'external', 'gateway']),
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
  const { getOrgUrl } = useOrganization()
  
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
      ingressRules: [],
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
        ingressRules: agent.spec.ingress?.map((rule: any) => ({
          description: rule.description || '',
          from: rule.from?.type || 'external',
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
        ingressRules: values.ingressRules,
      }

      await updateAgent.mutateAsync({ name: agentName, agent: formData as any })
      
      toast({
        title: 'Agent updated successfully',
        description: `Agent "${values.name}" has been updated.`,
      })
      
      // Redirect to agent detail page
      router.push(getOrgUrl(`/clusters/${clusterName}/agents/${agentName}`))
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
      <div className="space-y-6">
        <div className="h-96 bg-stone-200 dark:bg-stone-700 animate-pulse"></div>
      </div>
    )
  }

  if (!agent) {
    return (
      <div className="space-y-6">
        <div className="text-center py-8">
          <p className="text-stone-600 dark:text-stone-400">Agent not found</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
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
                <TabsContent value="basic" className="space-y-6 mt-3">
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
                              <div className="text-sm text-stone-600 dark:text-stone-400">Loading available models...</div>
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
                                      <div className="text-sm text-stone-600 dark:text-stone-400">{model.spec.provider} - {model.spec.modelName}</div>
                                    </div>
                                  </div>
                                ))}
                                {availableModels.length === 0 && (
                                  <div className="text-sm text-stone-600 dark:text-stone-400">No models available in this cluster</div>
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
                              <div className="text-sm text-stone-600 dark:text-stone-400">Loading available tools...</div>
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
                                  <div className="text-sm text-stone-600 dark:text-stone-400">No tools available in this cluster</div>
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
                <TabsContent value="resources" className="space-y-6 mt-3">
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
                <TabsContent value="networking" className="space-y-6 mt-3">
                  <ResourceNetworkPolicyForm
                    control={form.control}
                    egressRulesFieldName="egressRules"
                    ingressRulesFieldName="ingressRules"
                    watchedEgressRules={watchedValues.egressRules}
                    watchedIngressRules={watchedValues.ingressRules}
                    setValue={form.setValue}
                    getValues={form.getValues}
                    title="Network Policy"
                    description="Control network access for security. By default, agents can access cluster resources but no external endpoints."
                    resourceType="agent"
                    showIngressRules={true}
                  />
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
  )
}