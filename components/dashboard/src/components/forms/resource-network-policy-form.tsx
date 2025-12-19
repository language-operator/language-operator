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

interface ResourceNetworkPolicyFormProps<T extends FieldValues> {
  control: Control<T>
  egressRulesFieldName: Path<T>
  watchedEgressRules?: EgressRule[]
  setValue: (name: Path<T>, value: EgressRule[]) => void
  getValues: (name?: Path<T>) => EgressRule[]
  title?: string
  description?: string
  resourceType?: string
}

export function ResourceNetworkPolicyForm<T extends FieldValues>({
  control,
  egressRulesFieldName,
  watchedEgressRules = [],
  setValue,
  getValues,
  title = "Network Policy",
  description = "Control external network access for security. By default, resources can access cluster resources but no external endpoints.",
  resourceType = "resource"
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

  const commonRules = [
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
              {commonRules.map((template) => (
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
      </CardContent>
    </Card>
  )
}