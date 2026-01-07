'use client'

import { useEffect } from 'react'
import { useRouter, useParams } from 'next/navigation'
import Link from 'next/link'
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
  ArrowLeft, Bot, Save, Plus, X, 
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
import { useCreateAgent } from '@/hooks/use-agents'
import { useModels } from '@/hooks/use-models'
import { useTools } from '@/hooks/use-tools'
import { usePersonas } from '@/hooks/use-personas'
import { LanguageAgentFormData, LanguageAgent } from '@/types/agent'
import { useToast } from '@/hooks/use-toast'
import { kubernetesNameValidation } from '@/lib/validation'
import { useOrganization } from '@/components/organization-provider'

// Simplified form validation schema - only 5 essential fields
const agentFormSchema = z.object({
  instructions: z.string()
    .min(1, 'Goal is required')
    .min(10, 'Goal must be at least 10 characters')
    .max(5000, 'Goal must be less than 5000 characters'),
  name: kubernetesNameValidation,
  selectedModels: z.array(z.string()).min(1, 'At least one model must be selected'),
  selectedTools: z.array(z.string()),
  selectedPersona: z.string().optional(),
})

type AgentFormValues = z.infer<typeof agentFormSchema>

// Data fetching hooks - will be implemented next
// Models, tools, and personas will be fetched from cluster APIs

export default function CreateClusterAgentPage() {
  const router = useRouter()
  const params = useParams()
  const clusterName = params?.name as string
  
  const { toast } = useToast()
  const { getOrgUrl } = useOrganization()
  
  const createAgent = useCreateAgent(clusterName)
  
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
    mode: 'onChange',
    defaultValues: {
      instructions: '',
      name: '',
      selectedModels: availableModels.length > 0 ? [availableModels[0].metadata.name] : [],
      selectedTools: [],
      selectedPersona: 'none',
    },
  })

  const { formState } = form


  const onSubmit = async (values: AgentFormValues) => {
    try {
      // Get the first selected model for legacy API compatibility
      const firstModel = availableModels.find((m: any) => values.selectedModels.includes(m.metadata.name))
      console.log('First model data:', firstModel)
      
      const formData: LanguageAgentFormData = {
        instructions: values.instructions,
        name: values.name,
        namespace: clusterName ? `cluster-${clusterName}` : 'default',
        selectedModels: values.selectedModels,
        selectedTools: values.selectedTools,
        selectedPersona: values.selectedPersona === 'none' ? undefined : values.selectedPersona,
        // Legacy fields for API compatibility
        executionMode: 'autonomous',
        replicas: 1,
        modelName: values.selectedModels[0] || '',
        modelProvider: firstModel?.spec?.provider || '',
        modelEndpoint: firstModel?.spec?.endpoint || '',
        modelParameters: firstModel?.spec?.parameters || {},
        personaName: values.selectedPersona === 'none' ? '' : values.selectedPersona || '',
        personaInstructions: values.instructions,
        // Cluster reference for cluster-scoped agents
        clusterRef: clusterName || '',
      }

      console.log('Submitting form data:', formData)
      await createAgent.mutateAsync(formData)
      
      toast({
        title: 'Agent created successfully',
        description: `Agent "${values.name}" has been created and is starting up.`,
      })
      
      // Redirect to cluster agents page
      router.push(getOrgUrl(`/clusters/${clusterName}/agents`))
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
    <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <Button 
              variant="outline" 
              size="icon"
              onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/agents`))}
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <Bot className="h-8 w-8 text-blue-500" />
            <div>
              <h1 className="text-3xl font-bold">Create Language Agent</h1>
              <p className="text-muted-foreground">
                Configure and deploy a new language agent in the {clusterName} cluster
              </p>
            </div>
          </div>
        </div>

        <div className="grid gap-6">
          {/* Simplified Single-Column Form */}
          <div>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
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
                          <FormLabel className="text-lg font-semibold">Goal *</FormLabel>
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
                            <Input placeholder="my-agent" {...field} />
                          </FormControl>
                          <FormDescription>
                            Must be a valid Kubernetes resource name (lowercase, numbers, hyphens)
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
                          <FormDescription>
                            Optional personality configuration for your agent
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </CardContent>

                  {/* Form Actions */}
                </Card>

                {/* Form Actions */}
                <div className="flex items-center justify-between">
                  <Button 
                    type="button" 
                    variant="outline"
                    onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/agents`))}
                  >
                    Cancel
                  </Button>
                  
                  <Button 
                    type="submit" 
                    disabled={createAgent.isPending || !formState.isValid}
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

        </div>
      </div>
  )
}
