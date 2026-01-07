'use client'

import { useParams, useRouter } from 'next/navigation'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  ArrowLeft, 
  Download, 
  Shield, 
  Network, 
  Server, 
  AlertCircle,
  CheckCircle,
  Loader2
} from 'lucide-react'
import Link from 'next/link'
import { useEffect, useState } from 'react'
import { useOrganization } from '@/components/organization-provider'
import { ToolCatalogEntry } from '@/types/tool-catalog'
import { ResourceHeader } from '@/components/ui/resource-header'
import { Wrench } from 'lucide-react'
import { fetchWithOrganization } from '@/lib/api-client'

export default function InstallToolPage() {
  const params = useParams()
  const router = useRouter()
  const { getOrgUrl } = useOrganization()
  const clusterName = params?.name as string
  const toolName = params?.toolName as string
  
  const [tool, setTool] = useState<ToolCatalogEntry | null>(null)
  const [loading, setLoading] = useState(true)
  const [installing, setInstalling] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    fetchToolDetails()
  }, [toolName])

  const fetchToolDetails = async () => {
    try {
      setLoading(true)
      setError(null)

      const response = await fetch('/api/tools/catalog')
      if (!response.ok) {
        throw new Error('Failed to fetch tool catalog')
      }

      const catalog = await response.json()
      const toolData = catalog.tools[toolName]
      
      if (!toolData) {
        throw new Error(`Tool "${toolName}" not found in catalog`)
      }

      setTool(toolData)
    } catch (err) {
      console.error('Error fetching tool:', err)
      setError(err instanceof Error ? err.message : 'Failed to load tool details')
    } finally {
      setLoading(false)
    }
  }

  const handleInstall = async () => {
    if (!tool) return

    try {
      setInstalling(true)
      setError(null)

      const response = await fetchWithOrganization('/api/tools/install', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          toolId: toolName,
          clusterName: clusterName,
        }),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to install tool')
      }

      setSuccess(true)
      
      // Redirect after a short delay
      setTimeout(() => {
        router.push(getOrgUrl(`/clusters/${clusterName}/tools`))
      }, 2000)
    } catch (err) {
      console.error('Error installing tool:', err)
      setError(err instanceof Error ? err.message : 'Failed to install tool')
    } finally {
      setInstalling(false)
    }
  }

  if (loading) {
    return (
      <AuthenticatedLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-stone-900 dark:border-stone-100 mx-auto"></div>
            <p className="mt-4 text-stone-600 dark:text-stone-400">Loading tool details...</p>
          </div>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (error && !tool) {
    return (
      <AuthenticatedLayout>
        <div className="space-y-6">
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-16">
              <AlertCircle className="h-16 w-16 text-red-500 mb-4" />
              <CardTitle className="text-xl mb-2">Error Loading Tool</CardTitle>
              <CardDescription className="text-center max-w-md mb-4">
                {error}
              </CardDescription>
              <Button asChild variant="outline">
                <Link href={getOrgUrl(`/clusters/${clusterName}/tools`)}>
                  <ArrowLeft className="h-4 w-4 mr-2" />
                  Back to Tools
                </Link>
              </Button>
            </CardContent>
          </Card>
        </div>
      </AuthenticatedLayout>
    )
  }

  if (!tool) {
    return null
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <ResourceHeader
          backHref={getOrgUrl(`/clusters/${clusterName}/tools`)}
          backLabel="Back to Tools"
          icon={Wrench}
          title={`Install ${tool.displayName}`}
          subtitle={`Review tool details before installing to ${clusterName} cluster`}
        />

        {/* Success Message */}
        {success && (
          <Alert className="bg-green-50 border-green-200 dark:bg-green-950/50 dark:border-green-800">
            <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-400" />
            <AlertDescription className="text-green-800 dark:text-green-200">
              Tool installed successfully! Redirecting to tools list...
            </AlertDescription>
          </Alert>
        )}

        {/* Error Message */}
        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Tool Details */}
        <Card>
          <CardHeader>
            <CardTitle>{tool.displayName}</CardTitle>
            <CardDescription>{tool.description}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Basic Information */}
            <div>
              <h3 className="font-semibold mb-3">Basic Information</h3>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-stone-600 dark:text-stone-400">Tool ID:</span>
                  <span className="ml-2 font-mono">{toolName}</span>
                </div>
                <div>
                  <span className="text-stone-600 dark:text-stone-400">Type:</span>
                  <Badge variant="secondary" className="ml-2">
                    {tool.type.toUpperCase()}
                  </Badge>
                </div>
                <div>
                  <span className="text-stone-600 dark:text-stone-400">Deployment Mode:</span>
                  <Badge variant="outline" className="ml-2">
                    {tool.deploymentMode}
                  </Badge>
                </div>
                {tool.port && (
                  <div>
                    <span className="text-stone-600 dark:text-stone-400">Port:</span>
                    <span className="ml-2">{tool.port}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Container */}
            <div>
              <h3 className="font-semibold mb-3">Container</h3>
              <div className="flex items-center gap-2 p-3 bg-stone-50 dark:bg-stone-900 border border-stone-200 dark:border-stone-800 rounded-md">
                <Server className="h-4 w-4 text-stone-600 dark:text-stone-400" />
                <code className="text-sm flex-1 font-mono">{tool.image}</code>
              </div>
            </div>

            {/* Security & Permissions */}
            <div>
              <h3 className="font-semibold mb-3">Security & Permissions</h3>
              <div className="space-y-3">
                {tool.authRequired && (
                  <div className="flex items-center gap-2 text-sm">
                    <Shield className="h-4 w-4 text-amber-600" />
                    <span>Authentication required for this tool</span>
                  </div>
                )}
                
                {tool.rbac && (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm">
                      <Shield className="h-4 w-4 text-blue-600" />
                      <span>RBAC permissions will be configured</span>
                    </div>
                    {tool.rbac.clusterRole?.rules.map((rule, idx) => (
                      <div key={idx} className="ml-6 p-2 bg-stone-50 dark:bg-stone-900 border border-stone-200 dark:border-stone-800 rounded text-xs font-mono">
                        <div>API Groups: {rule.apiGroups.join(', ')}</div>
                        <div>Resources: {rule.resources.join(', ')}</div>
                        <div>Verbs: {rule.verbs.join(', ')}</div>
                      </div>
                    ))}
                  </div>
                )}

                {tool.egress && tool.egress.length > 0 && (
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm">
                      <Network className="h-4 w-4 text-purple-600" />
                      <span>Network policies will be applied</span>
                    </div>
                    {tool.egress.map((rule, idx) => (
                      <div key={idx} className="ml-6 p-2 bg-stone-50 dark:bg-stone-900 border border-stone-200 dark:border-stone-800 rounded text-xs">
                        <div className="font-medium">{rule.description}</div>
                        {rule.dns && (
                          <div>DNS: {rule.dns.join(', ')}</div>
                        )}
                        {rule.ports && (
                          <div>
                            Ports: {rule.ports.map(p => `${p.port}/${p.protocol}`).join(', ')}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-end gap-3 pt-4 border-t">
              <Button
                variant="outline"
                onClick={() => router.push(getOrgUrl(`/clusters/${clusterName}/tools`))}
                disabled={installing}
              >
                Cancel
              </Button>
              <Button 
                onClick={handleInstall} 
                disabled={installing || success}
              >
                {installing ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Installing...
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-2" />
                    Install Tool
                  </>
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}