'use client'

import React from 'react'
import { Control, FieldPath, FieldValues, Path } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { 
  Plus, Minus, Network, Shield, ArrowLeft, ArrowRight
} from 'lucide-react'
import {
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from '@/components/ui/form'

export interface EgressRule {
  description?: string
  dns?: string[]
  cidr?: string
  ports?: Array<{
    port: number
    protocol: 'TCP' | 'UDP'
  }>
}

export interface IngressRule {
  description?: string
  from: 'agents' | 'tools' | 'models' | 'cluster' | 'external' | 'gateway'
  ports?: Array<{
    port: number
    protocol: 'TCP' | 'UDP'
  }>
}

interface EnhancedNetworkPolicyFormProps<T extends FieldValues> {
  control: Control<T>
  egressRulesFieldName: Path<T>
  ingressRulesFieldName?: Path<T>
  watchedEgressRules?: EgressRule[]
  watchedIngressRules?: IngressRule[]
  setValue: (name: Path<T>, value: any) => void
  getValues: (name?: Path<T>) => any
  title?: string
  description?: string
  resourceType?: string
  showIngressRules?: boolean
}

export function EnhancedNetworkPolicyForm<T extends FieldValues>({
  control,
  egressRulesFieldName,
  ingressRulesFieldName,
  watchedEgressRules = [],
  watchedIngressRules = [],
  setValue,
  getValues,
  title = "Network Policy",
  description = "Control network access for security and compliance.",
  resourceType = "resource",
  showIngressRules = false
}: EnhancedNetworkPolicyFormProps<T>) {

  // Egress rule management
  const addEgressRule = () => {
    const currentRules = getValues(egressRulesFieldName) || []
    const newRule: EgressRule = {
      description: '',
      dns: [],
      cidr: '',
      ports: []
    }
    setValue(egressRulesFieldName, [...currentRules, newRule])
  }

  const removeEgressRule = (index: number) => {
    const currentRules = getValues(egressRulesFieldName) || []
    const newRules = currentRules.filter((_: any, i: number) => i !== index)
    setValue(egressRulesFieldName, newRules)
  }

  // Ingress rule management
  const addIngressRule = () => {
    if (!ingressRulesFieldName) return
    const currentRules = getValues(ingressRulesFieldName) || []
    const newRule: IngressRule = {
      description: '',
      from: 'agents',
      ports: []
    }
    setValue(ingressRulesFieldName, [...currentRules, newRule])
  }

  const removeIngressRule = (index: number) => {
    if (!ingressRulesFieldName) return
    const currentRules = getValues(ingressRulesFieldName) || []
    const newRules = currentRules.filter((_: any, i: number) => i !== index)
    setValue(ingressRulesFieldName, newRules)
  }

  // Common rule templates
  const commonEgressRules = [
    {
      name: 'OpenAI',
      rule: {
        description: 'Allow OpenAI API access',
        dns: ['api.openai.com'],
        cidr: '',
        ports: [{ port: 443, protocol: 'TCP' as const }]
      }
    },
    {
      name: 'Anthropic',
      rule: {
        description: 'Allow Anthropic API access', 
        dns: ['api.anthropic.com'],
        cidr: '',
        ports: [{ port: 443, protocol: 'TCP' as const }]
      }
    }
  ]

  const commonIngressRules = [
    {
      name: 'From Agents',
      rule: {
        description: 'Allow access from language agents',
        from: 'agents' as const,
        ports: [{ port: 8080, protocol: 'TCP' as const }]
      }
    },
    {
      name: 'External Access',
      rule: {
        description: 'Allow external webhook access',
        from: 'external' as const,
        ports: [{ port: 8080, protocol: 'TCP' as const }]
      }
    }
  ]

  const addCommonEgressRule = (ruleTemplate: EgressRule) => {
    const currentRules = getValues(egressRulesFieldName) || []
    setValue(egressRulesFieldName, [...currentRules, ruleTemplate])
  }

  const addCommonIngressRule = (ruleTemplate: IngressRule) => {
    if (!ingressRulesFieldName) return
    const currentRules = getValues(ingressRulesFieldName) || []
    setValue(ingressRulesFieldName, [...currentRules, ruleTemplate])
  }

  const getIngressRuleDescription = (from: string) => {
    switch (from) {
      case 'agents': return 'Language agents in this cluster'
      case 'tools': return 'Language tools in this cluster'
      case 'models': return 'Language models in this cluster'
      case 'cluster': return 'Any resource managed by Language Operator'
      case 'external': return 'External services and webhook callers'
      case 'gateway': return 'Gateway controllers (nginx, envoy, etc.)'
      default: return 'Unknown source'
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-5 w-5" />
          {title}
        </CardTitle>
        <CardDescription>
          {description}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {showIngressRules ? (
          <Tabs defaultValue="egress" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="egress" className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4" />
                Egress (Outbound)
              </TabsTrigger>
              <TabsTrigger value="ingress" className="flex items-center gap-2">
                <ArrowLeft className="h-4 w-4" />
                Ingress (Inbound)
              </TabsTrigger>
            </TabsList>

            {/* Egress Tab Content */}
            <TabsContent value="egress" className="space-y-4">
              <Alert>
                <ArrowRight className="h-4 w-4" />
                <AlertDescription>
                  Egress rules control which external endpoints this {resourceType} can access. 
                  Cluster-internal communication is always allowed.
                </AlertDescription>
              </Alert>

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <Label className="text-base font-medium">Egress Rules</Label>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={addEgressRule}
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    Add Rule
                  </Button>
                </div>
                
                {(!watchedEgressRules || watchedEgressRules.length === 0) ? (
                  <div className="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
                    <ArrowRight className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p>No egress rules configured</p>
                    <p className="text-sm">External access is blocked by default</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {watchedEgressRules.map((rule, index) => (
                      <Card key={index} className="border-l-4 border-l-orange-500">
                        <CardContent className="pt-4">
                          <div className="space-y-4">
                            <div className="flex items-start justify-between">
                              <div className="space-y-2 flex-1">
                                <Label>Description</Label>
                                <FormField
                                  control={control}
                                  name={`${egressRulesFieldName}.${index}.description` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input
                                          placeholder="e.g., Allow OpenAI API access"
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
                                variant="outline"
                                size="sm"
                                className="ml-2"
                                onClick={() => removeEgressRule(index)}
                              >
                                <Minus className="h-4 w-4" />
                              </Button>
                            </div>
                            
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label>DNS Names</Label>
                                <FormField
                                  control={control}
                                  name={`${egressRulesFieldName}.${index}.dns` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input
                                          placeholder="api.openai.com, *.azure.com"
                                          className="font-mono text-sm"
                                          value={Array.isArray(field.value) ? field.value.join(', ') : ''}
                                          onChange={(e) => {
                                            const dnsNames = e.target.value
                                              .split(',')
                                              .map(name => name.trim())
                                              .filter(name => name.length > 0)
                                            field.onChange(dnsNames)
                                          }}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </div>
                              
                              <div className="space-y-2">
                                <Label>CIDR Block</Label>
                                <FormField
                                  control={control}
                                  name={`${egressRulesFieldName}.${index}.cidr` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input
                                          placeholder="192.168.1.0/24"
                                          className="font-mono text-sm"
                                          {...field}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </div>
                            </div>
                            
                            <div className="space-y-2">
                              <Label>Ports</Label>
                              <FormField
                                control={control}
                                name={`${egressRulesFieldName}.${index}.ports` as Path<T>}
                                render={({ field }) => (
                                  <FormItem>
                                    <FormControl>
                                      <Input
                                        placeholder="443, 80"
                                        value={
                                          Array.isArray(field.value) 
                                            ? field.value.map((p: any) => typeof p === 'object' ? p.port : p).join(', ')
                                            : ''
                                        }
                                        onChange={(e) => {
                                          const ports = e.target.value
                                            .split(',')
                                            .map(port => parseInt(port.trim()))
                                            .filter(port => !isNaN(port) && port > 0 && port < 65536)
                                            .map(port => ({ port, protocol: 'TCP' as const }))
                                          field.onChange(ports)
                                        }}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}

                <div className="space-y-3">
                  <Label className="text-sm font-medium">Common Patterns</Label>
                  <div className="flex gap-2 flex-wrap">
                    {commonEgressRules.map((template) => (
                      <Button
                        key={template.name}
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => addCommonEgressRule(template.rule)}
                      >
                        + {template.name}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            </TabsContent>

            {/* Ingress Tab Content */}
            <TabsContent value="ingress" className="space-y-4">
              <Alert>
                <ArrowLeft className="h-4 w-4" />
                <AlertDescription>
                  Ingress rules control which services can connect to this {resourceType}. 
                  Configure these to enable specific communication patterns.
                </AlertDescription>
              </Alert>

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <Label className="text-base font-medium">Ingress Rules</Label>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={addIngressRule}
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    Add Rule
                  </Button>
                </div>
                
                {(!watchedIngressRules || watchedIngressRules.length === 0) ? (
                  <div className="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
                    <ArrowLeft className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p>No ingress rules configured</p>
                    <p className="text-sm">Inbound access is determined by default policies</p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    {watchedIngressRules.map((rule, index) => (
                      <Card key={index} className="border-l-4 border-l-blue-500">
                        <CardContent className="pt-4">
                          <div className="space-y-4">
                            <div className="flex items-start justify-between">
                              <div className="space-y-2 flex-1">
                                <Label>Description</Label>
                                <FormField
                                  control={control}
                                  name={`${ingressRulesFieldName}.${index}.description` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input
                                          placeholder="e.g., Allow access from agents"
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
                                variant="outline"
                                size="sm"
                                className="ml-2"
                                onClick={() => removeIngressRule(index)}
                              >
                                <Minus className="h-4 w-4" />
                              </Button>
                            </div>
                            
                            <div className="grid grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <Label>Allow Access From</Label>
                                <FormField
                                  control={control}
                                  name={`${ingressRulesFieldName}.${index}.from` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <Select onValueChange={field.onChange} defaultValue={field.value}>
                                        <FormControl>
                                          <SelectTrigger>
                                            <SelectValue placeholder="Select source" />
                                          </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                          <SelectItem value="agents">
                                            <div className="flex flex-col">
                                              <span>Agents</span>
                                              <span className="text-xs text-muted-foreground">Language agents in this cluster</span>
                                            </div>
                                          </SelectItem>
                                          <SelectItem value="tools">
                                            <div className="flex flex-col">
                                              <span>Tools</span>
                                              <span className="text-xs text-muted-foreground">Language tools in this cluster</span>
                                            </div>
                                          </SelectItem>
                                          <SelectItem value="models">
                                            <div className="flex flex-col">
                                              <span>Models</span>
                                              <span className="text-xs text-muted-foreground">Language models in this cluster</span>
                                            </div>
                                          </SelectItem>
                                          <SelectItem value="cluster">
                                            <div className="flex flex-col">
                                              <span>Cluster Resources</span>
                                              <span className="text-xs text-muted-foreground">Any Language Operator resource</span>
                                            </div>
                                          </SelectItem>
                                          <SelectItem value="external">
                                            <div className="flex flex-col">
                                              <span>External</span>
                                              <span className="text-xs text-muted-foreground">External services and webhooks</span>
                                            </div>
                                          </SelectItem>
                                          <SelectItem value="gateway">
                                            <div className="flex flex-col">
                                              <span>Gateway</span>
                                              <span className="text-xs text-muted-foreground">Gateway controllers (nginx, envoy)</span>
                                            </div>
                                          </SelectItem>
                                        </SelectContent>
                                      </Select>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                                {rule.from && (
                                  <p className="text-xs text-muted-foreground">
                                    {getIngressRuleDescription(rule.from)}
                                  </p>
                                )}
                              </div>
                              
                              <div className="space-y-2">
                                <Label>Ports</Label>
                                <FormField
                                  control={control}
                                  name={`${ingressRulesFieldName}.${index}.ports` as Path<T>}
                                  render={({ field }) => (
                                    <FormItem>
                                      <FormControl>
                                        <Input
                                          placeholder="8080, 443"
                                          value={
                                            Array.isArray(field.value) 
                                              ? field.value.map((p: any) => typeof p === 'object' ? p.port : p).join(', ')
                                              : ''
                                          }
                                          onChange={(e) => {
                                            const ports = e.target.value
                                              .split(',')
                                              .map(port => parseInt(port.trim()))
                                              .filter(port => !isNaN(port) && port > 0 && port < 65536)
                                              .map(port => ({ port, protocol: 'TCP' as const }))
                                            field.onChange(ports)
                                          }}
                                        />
                                      </FormControl>
                                      <FormMessage />
                                    </FormItem>
                                  )}
                                />
                              </div>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}

                <div className="space-y-3">
                  <Label className="text-sm font-medium">Common Patterns</Label>
                  <div className="flex gap-2 flex-wrap">
                    {commonIngressRules.map((template) => (
                      <Button
                        key={template.name}
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => addCommonIngressRule(template.rule)}
                      >
                        + {template.name}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            </TabsContent>
          </Tabs>
        ) : (
          /* Fallback to egress-only for backward compatibility */
          <div className="space-y-4">
            <Alert>
              <Network className="h-4 w-4" />
              <AlertDescription>
                Egress rules control which external endpoints this {resourceType} can access. 
                Cluster-internal communication is always allowed.
              </AlertDescription>
            </Alert>
            
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Label className="text-base font-medium">Egress Rules</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addEgressRule}
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add Rule
                </Button>
              </div>
              
              {(!watchedEgressRules || watchedEgressRules.length === 0) ? (
                <div className="text-center py-8 text-muted-foreground border-2 border-dashed rounded-lg">
                  <Network className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No egress rules configured</p>
                  <p className="text-sm">External access is blocked by default</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {watchedEgressRules.map((rule, index) => (
                    <Card key={index} className="border-l-4 border-l-orange-500">
                      <CardContent className="pt-4">
                        <div className="space-y-4">
                          <div className="flex items-start justify-between">
                            <div className="space-y-2 flex-1">
                              <Label>Description</Label>
                              <FormField
                                control={control}
                                name={`${egressRulesFieldName}.${index}.description` as Path<T>}
                                render={({ field }) => (
                                  <FormItem>
                                    <FormControl>
                                      <Input
                                        placeholder="e.g., Allow OpenAI API access"
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
                              variant="outline"
                              size="sm"
                              className="ml-2"
                              onClick={() => removeEgressRule(index)}
                            >
                              <Minus className="h-4 w-4" />
                            </Button>
                          </div>
                          
                          <div className="grid grid-cols-2 gap-4">
                            <div className="space-y-2">
                              <Label>DNS Names</Label>
                              <FormField
                                control={control}
                                name={`${egressRulesFieldName}.${index}.dns` as Path<T>}
                                render={({ field }) => (
                                  <FormItem>
                                    <FormControl>
                                      <Input
                                        placeholder="api.openai.com, *.azure.com"
                                        className="font-mono text-sm"
                                        value={Array.isArray(field.value) ? field.value.join(', ') : ''}
                                        onChange={(e) => {
                                          const dnsNames = e.target.value
                                            .split(',')
                                            .map(name => name.trim())
                                            .filter(name => name.length > 0)
                                          field.onChange(dnsNames)
                                        }}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            </div>
                            
                            <div className="space-y-2">
                              <Label>CIDR Block</Label>
                              <FormField
                                control={control}
                                name={`${egressRulesFieldName}.${index}.cidr` as Path<T>}
                                render={({ field }) => (
                                  <FormItem>
                                    <FormControl>
                                      <Input
                                        placeholder="192.168.1.0/24"
                                        className="font-mono text-sm"
                                        {...field}
                                      />
                                    </FormControl>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}

              <div className="space-y-3">
                <Label className="text-sm font-medium">Common Patterns</Label>
                <div className="flex gap-2 flex-wrap">
                  {commonEgressRules.map((template) => (
                    <Button
                      key={template.name}
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => addCommonEgressRule(template.rule)}
                    >
                      + {template.name}
                    </Button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}