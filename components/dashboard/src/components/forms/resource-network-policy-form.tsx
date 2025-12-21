'use client'

import React from 'react'
import { Control, FieldPath, FieldValues, Path } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Plus, Trash2, Network, Shield, Minus
} from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
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

interface ResourceNetworkPolicyFormProps<T extends FieldValues> {
  control: Control<T>
  egressRulesFieldName: Path<T>
  ingressRulesFieldName?: Path<T>
  watchedEgressRules?: EgressRule[]
  watchedIngressRules?: IngressRule[]
  setValue: (name: Path<T>, value: EgressRule[] | IngressRule[]) => void
  getValues: (name?: Path<T>) => EgressRule[] | IngressRule[]
  title?: string
  description?: string
  resourceType?: string
  showIngressRules?: boolean
}

export function ResourceNetworkPolicyForm<T extends FieldValues>({
  control,
  egressRulesFieldName,
  ingressRulesFieldName,
  watchedEgressRules = [],
  watchedIngressRules = [],
  setValue,
  getValues,
  title = "Network Policy",
  description = "Control external network access for security. By default, resources can access cluster resources but no external endpoints.",
  resourceType = "resource",
  showIngressRules = false
}: ResourceNetworkPolicyFormProps<T>) {

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
    const newRules = currentRules.filter((_, i) => i !== index)
    setValue(egressRulesFieldName, newRules)
  }

  const addCommonRule = (ruleTemplate: EgressRule) => {
    const currentRules = getValues(egressRulesFieldName) || []
    setValue(egressRulesFieldName, [...currentRules, ruleTemplate])
  }

  // Ingress rule management functions
  const addIngressRule = () => {
    if (!ingressRulesFieldName) return
    const currentRules = (getValues(ingressRulesFieldName) as IngressRule[]) || []
    const newRule: IngressRule = {
      description: '',
      from: 'agents',
      ports: []
    }
    setValue(ingressRulesFieldName, [...currentRules, newRule])
  }

  const removeIngressRule = (index: number) => {
    if (!ingressRulesFieldName) return
    const currentRules = (getValues(ingressRulesFieldName) as IngressRule[]) || []
    const newRules = currentRules.filter((_, i) => i !== index)
    setValue(ingressRulesFieldName, newRules)
  }

  const addCommonIngressRule = (ruleTemplate: IngressRule) => {
    if (!ingressRulesFieldName) return
    const currentRules = (getValues(ingressRulesFieldName) as IngressRule[]) || []
    setValue(ingressRulesFieldName, [...currentRules, ruleTemplate])
  }

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
    },
    {
      name: 'Google APIs',
      rule: {
        description: 'Allow Google APIs access',
        dns: ['*.googleapis.com'],
        cidr: '',
        ports: [{ port: 443, protocol: 'TCP' as const }]
      }
    },
    {
      name: 'Azure OpenAI',
      rule: {
        description: 'Allow Azure OpenAI access',
        dns: ['*.openai.azure.com'],
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
      name: 'From Tools',
      rule: {
        description: 'Allow access from language tools',
        from: 'tools' as const,
        ports: [{ port: 8080, protocol: 'TCP' as const }]
      }
    },
    {
      name: 'From Models',
      rule: {
        description: 'Allow access from language models',
        from: 'models' as const,
        ports: [{ port: 8080, protocol: 'TCP' as const }]
      }
    },
    {
      name: 'From Cluster',
      rule: {
        description: 'Allow access from any cluster resource',
        from: 'cluster' as const,
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
    },
    {
      name: 'Gateway Access',
      rule: {
        description: 'Allow access from gateway controllers',
        from: 'gateway' as const,
        ports: [{ port: 8080, protocol: 'TCP' as const }]
      }
    }
  ]

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
              <TabsTrigger value="egress">Egress Rules</TabsTrigger>
              <TabsTrigger value="ingress">Ingress Rules</TabsTrigger>
            </TabsList>
            <TabsContent value="egress" className="space-y-4">
              <Alert>
                <Network className="h-4 w-4" />
                <AlertDescription>
                  Egress rules control which external endpoints this {resourceType} can access. 
                  Only configure rules for external APIs - cluster-internal communication is always allowed.
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
              <p className="text-sm">Click "Add Rule" to configure external access</p>
            </div>
          ) : (
            <div className="space-y-4">
              {watchedEgressRules.map((rule, index) => (
                <Card key={index} className="border-l-4 border-l-blue-500">
                  <CardContent className="pt-4">
                    <div className="space-y-4">
                      <div className="flex items-start justify-between">
                        <div className="space-y-2 flex-1">
                          <Label htmlFor={`rule-description-${index}`}>Description</Label>
                          <FormField
                            control={control}
                            name={`${egressRulesFieldName}.${index}.description` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Input
                                    id={`rule-description-${index}`}
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
                          <Label htmlFor={`rule-dns-${index}`}>DNS Names</Label>
                          <FormField
                            control={control}
                            name={`${egressRulesFieldName}.${index}.dns` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Input
                                    id={`rule-dns-${index}`}
                                    placeholder="api.openai.com, *.azure.com"
                                    className="font-mono"
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
                                <p className="text-xs text-muted-foreground">
                                  Comma-separated. Use * for wildcards.
                                </p>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                        
                        <div className="space-y-2">
                          <Label htmlFor={`rule-cidr-${index}`}>CIDR Block</Label>
                          <FormField
                            control={control}
                            name={`${egressRulesFieldName}.${index}.cidr` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Input
                                    id={`rule-cidr-${index}`}
                                    placeholder="192.168.1.0/24"
                                    className="font-mono"
                                    {...field}
                                  />
                                </FormControl>
                                <p className="text-xs text-muted-foreground">
                                  For local networks. Use DNS for cloud APIs.
                                </p>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </div>
                      
                      <div className="space-y-2">
                        <Label htmlFor={`rule-ports-${index}`}>Ports</Label>
                        <FormField
                          control={control}
                          name={`${egressRulesFieldName}.${index}.ports` as Path<T>}
                          render={({ field }) => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  id={`rule-ports-${index}`}
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
                              <p className="text-xs text-muted-foreground">
                                Comma-separated port numbers. Use 443 for HTTPS, 80 for HTTP.
                              </p>
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
        </div>

        <div className="space-y-4">
          <Separator />
          <div className="space-y-3">
            <Label className="text-sm font-medium">Common Patterns</Label>
            <div className="grid grid-cols-2 gap-2">
              {commonEgressRules.map((template) => (
                <Button
                  key={template.name}
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => addCommonRule(template.rule)}
                >
                  + {template.name}
                </Button>
              ))}
            </div>
          </div>
        </div>
            </TabsContent>
            <TabsContent value="ingress" className="space-y-4">
              <Alert>
                <Network className="h-4 w-4" />
                <AlertDescription>
                  Ingress rules control which sources can access this {resourceType}.
                  Configure rules to allow specific types of connections.
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
              <Network className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p>No ingress rules configured</p>
              <p className="text-sm">Click "Add Rule" to configure incoming access</p>
            </div>
          ) : (
            <div className="space-y-4">
              {watchedIngressRules.map((rule, index) => (
                <Card key={index} className="border-l-4 border-l-green-500">
                  <CardContent className="pt-4">
                    <div className="space-y-4">
                      <div className="flex items-start justify-between">
                        <div className="space-y-2 flex-1">
                          <Label htmlFor={`ingress-rule-description-${index}`}>Description</Label>
                          <FormField
                            control={control}
                            name={`${ingressRulesFieldName}.${index}.description` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Input
                                    id={`ingress-rule-description-${index}`}
                                    placeholder="e.g., Allow access from language agents"
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
                          <Label htmlFor={`ingress-rule-from-${index}`}>From</Label>
                          <FormField
                            control={control}
                            name={`${ingressRulesFieldName}.${index}.from` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <Select onValueChange={field.onChange} value={field.value as string}>
                                  <FormControl>
                                    <SelectTrigger>
                                      <SelectValue placeholder="Select source..." />
                                    </SelectTrigger>
                                  </FormControl>
                                  <SelectContent>
                                    <SelectItem value="agents">Language Agents</SelectItem>
                                    <SelectItem value="tools">Language Tools</SelectItem>
                                    <SelectItem value="models">Language Models</SelectItem>
                                    <SelectItem value="cluster">Any Cluster Resource</SelectItem>
                                    <SelectItem value="external">External (Internet)</SelectItem>
                                    <SelectItem value="gateway">Gateway Controllers</SelectItem>
                                  </SelectContent>
                                </Select>
                                <p className="text-xs text-muted-foreground">
                                  Choose the source type that can connect to this resource.
                                </p>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                        
                        <div className="space-y-2">
                          <Label htmlFor={`ingress-rule-ports-${index}`}>Ports</Label>
                          <FormField
                            control={control}
                            name={`${ingressRulesFieldName}.${index}.ports` as Path<T>}
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Input
                                    id={`ingress-rule-ports-${index}`}
                                    placeholder="8080, 8443"
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
                                <p className="text-xs text-muted-foreground">
                                  Comma-separated port numbers. Use 8080 for HTTP services.
                                </p>
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
        </div>

        <div className="space-y-4">
          <Separator />
          <div className="space-y-3">
            <Label className="text-sm font-medium">Common Patterns</Label>
            <div className="grid grid-cols-2 gap-2">
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
          <>
            <Alert>
              <Network className="h-4 w-4" />
              <AlertDescription>
                Egress rules control which external endpoints this {resourceType} can access. 
                Only configure rules for external APIs - cluster-internal communication is always allowed.
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
            <p className="text-sm">Click "Add Rule" to configure external access</p>
          </div>
        ) : (
          <div className="space-y-4">
            {watchedEgressRules.map((rule, index) => (
              <Card key={index} className="border-l-4 border-l-blue-500">
                <CardContent className="pt-4">
                  <div className="space-y-4">
                    <div className="flex items-start justify-between">
                      <div className="space-y-2 flex-1">
                        <Label htmlFor={`rule-description-${index}`}>Description</Label>
                        <FormField
                          control={control}
                          name={`${egressRulesFieldName}.${index}.description` as Path<T>}
                          render={({ field }) => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  id={`rule-description-${index}`}
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
                        <Label htmlFor={`rule-dns-${index}`}>DNS Names</Label>
                        <FormField
                          control={control}
                          name={`${egressRulesFieldName}.${index}.dns` as Path<T>}
                          render={({ field }) => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  id={`rule-dns-${index}`}
                                  placeholder="api.openai.com, *.azure.com"
                                  className="font-mono"
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
                              <p className="text-xs text-muted-foreground">
                                Comma-separated. Use * for wildcards.
                              </p>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                      
                      <div className="space-y-2">
                        <Label htmlFor={`rule-cidr-${index}`}>CIDR Block</Label>
                        <FormField
                          control={control}
                          name={`${egressRulesFieldName}.${index}.cidr` as Path<T>}
                          render={({ field }) => (
                            <FormItem>
                              <FormControl>
                                <Input
                                  id={`rule-cidr-${index}`}
                                  placeholder="192.168.1.0/24"
                                  className="font-mono"
                                  {...field}
                                />
                              </FormControl>
                              <p className="text-xs text-muted-foreground">
                                For local networks. Use DNS for cloud APIs.
                              </p>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                    </div>
                    
                    <div className="space-y-2">
                      <Label htmlFor={`rule-ports-${index}`}>Ports</Label>
                      <FormField
                        control={control}
                        name={`${egressRulesFieldName}.${index}.ports` as Path<T>}
                        render={({ field }) => (
                          <FormItem>
                            <FormControl>
                              <Input
                                id={`rule-ports-${index}`}
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
                            <p className="text-xs text-muted-foreground">
                              Comma-separated port numbers. Use 443 for HTTPS, 80 for HTTP.
                            </p>
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
      </div>

      <div className="space-y-4">
        <Separator />
        <div className="space-y-3">
          <Label className="text-sm font-medium">Common Patterns</Label>
          <div className="grid grid-cols-2 gap-2">
            {commonEgressRules.map((template) => (
              <Button
                key={template.name}
                type="button"
                variant="outline"
                size="sm"
                onClick={() => addCommonRule(template.rule)}
              >
                + {template.name}
              </Button>
            ))}
          </div>
        </div>
      </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}