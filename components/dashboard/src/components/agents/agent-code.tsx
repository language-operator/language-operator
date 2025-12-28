'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { AlertCircle, Code, History, Lock, Unlock, RotateCcw, Brain, Loader2 } from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { useAgentVersions, useRollbackAgent, useToggleAgentLock, useTriggerOptimization } from '@/hooks/use-agents'
import { LanguageAgent } from '@/types/agent'
import { formatTimeAgo } from './utils'

interface AgentCodeProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentCode({ agent, clusterName }: AgentCodeProps) {
  const [selectedVersionName, setSelectedVersionName] = useState<string>('')
  const [showRollbackDialog, setShowRollbackDialog] = useState(false)
  const [lockOnRollback, setLockOnRollback] = useState(false)
  const { theme } = useTheme()

  // Hooks for version management
  const { data: versionsResponse, isLoading: versionsLoading, error: versionsError } = useAgentVersions(agent.metadata.name || '', clusterName || '')
  const rollbackMutation = useRollbackAgent(clusterName || '')
  const lockMutation = useToggleAgentLock(clusterName || '')
  const optimizeMutation = useTriggerOptimization(clusterName || '')

  const versions = versionsResponse?.data || []
  const currentVersionName = versionsResponse?.currentVersion
  const isLocked = versionsResponse?.isLocked || false

  // Set initial selected version to current version
  useEffect(() => {
    if (currentVersionName && !selectedVersionName) {
      setSelectedVersionName(currentVersionName)
    }
  }, [currentVersionName, selectedVersionName])

  const selectedVersion = versions.find((v: any) => v.metadata.name === selectedVersionName)
  const synthesisInfo = agent.status?.synthesisInfo
  const isSynthesized = agent.status?.conditions?.some(
    (condition: any) => condition.type === 'Synthesized' && condition.status === 'True'
  )

  const handleRollback = async () => {
    if (!selectedVersionName || selectedVersionName === currentVersionName) return

    try {
      await rollbackMutation.mutateAsync({
        agentName: agent.metadata.name || '',
        versionName: selectedVersionName,
        lock: lockOnRollback
      })
      setShowRollbackDialog(false)
      setLockOnRollback(false)
    } catch (error) {
      console.error('Rollback failed:', error)
    }
  }

  const handleToggleLock = async () => {
    try {
      await lockMutation.mutateAsync({
        agentName: agent.metadata.name || '',
        lock: !isLocked
      })
    } catch (error) {
      console.error('Lock toggle failed:', error)
    }
  }

  const handleOptimize = async () => {
    try {
      await optimizeMutation.mutateAsync({
        agentName: agent.metadata.name || '',
      })
    } catch (error) {
      console.error('Optimization failed:', error)
    }
  }

  // Local formatTimeAgo for condensed version display
  const formatTimeAgoCondensed = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (days > 0) return `${days}d ago`
    if (hours > 0) return `${hours}h ago`
    if (minutes > 0) return `${minutes}m ago`
    return 'Just now'
  }

  if (versionsLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading agent versions...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Version Selector and Controls */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            Versions
          </CardTitle>
          <CardDescription>
            View and manage different versions of this agent's synthesized code
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-4 mb-4">
            {/* Version Selector */}
            <div className="flex-1">
              <Select
                value={selectedVersionName}
                onValueChange={setSelectedVersionName}
                disabled={versions.length === 0}
              >
                <SelectTrigger className="min-w-80">
                  <SelectValue placeholder="Select a version" />
                </SelectTrigger>
                <SelectContent>
                  {versions.map((version: any) => (
                    <SelectItem key={version.metadata.name} value={version.metadata.name}>
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-sm">v{version.spec.version}</span>
                        {version.isCurrent && (
                          <Badge variant="secondary" className="text-xs">CURRENT</Badge>
                        )}
                        <Badge variant="secondary" className="text-xs">
                          {version.spec.sourceType || 'manual'}
                        </Badge>
                        <span className="text-muted-foreground text-xs">
                          {formatTimeAgoCondensed(version.metadata.creationTimestamp)}
                        </span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Version Controls */}
            <div className="flex items-center gap-2">
              {/* Lock Toggle */}
              {currentVersionName && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleToggleLock}
                  disabled={lockMutation.isPending}
                  className={isLocked ? 'text-orange-600' : ''}
                >
                  {isLocked ? (
                    <>
                      <Lock className="h-4 w-4 mr-2" />
                      Unlock
                    </>
                  ) : (
                    <>
                      <Unlock className="h-4 w-4 mr-2" />
                      Lock
                    </>
                  )}
                </Button>
              )}

              {/* Optimize Button */}
              <Button
                variant="default"
                size="sm"
                onClick={handleOptimize}
                disabled={optimizeMutation.isPending}
              >
                {optimizeMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Optimizing...
                  </>
                ) : (
                  <>
                    <Brain className="h-4 w-4 mr-2" />
                    Optimize
                  </>
                )}
              </Button>

              {/* Rollback Button */}
              {selectedVersionName && selectedVersionName !== currentVersionName && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={() => setShowRollbackDialog(true)}
                  disabled={rollbackMutation.isPending}
                >
                  <RotateCcw className="h-4 w-4 mr-2" />
                  Roll Back to This Version
                </Button>
              )}
            </div>
          </div>

          {/* Status Indicators */}
          {isLocked && (
            <div className="flex items-center gap-1 text-orange-600 text-sm">
              <Lock className="h-3 w-3" />
              <span>Version Locked</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Synthesized Code */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Code className="h-5 w-5" />
            Agent Code {selectedVersion?.isCurrent ? '(Current)' : '(Version ' + selectedVersion?.spec?.version + ')'}
          </CardTitle>
          <CardDescription>
            Tasks and execution logic synthesized from agent instructions
          </CardDescription>
        </CardHeader>
        <CardContent>
          {versionsError ? (
            <div className="text-center py-16">
              <AlertCircle className="h-16 w-16 text-red-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2 text-red-600">Error Loading Versions</h3>
              <p className="text-muted-foreground max-w-md mx-auto">{versionsError.message}</p>
            </div>
          ) : selectedVersion?.spec?.code ? (
            <div className="border rounded-lg">
              <div className="bg-muted p-3 border-b">
                <div className="flex items-center justify-between">
                  <p className="font-medium text-sm">Ruby</p>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">v{selectedVersion.spec.version}</Badge>
                    {selectedVersion.isCurrent && (
                      <Badge variant="default">Current</Badge>
                    )}
                  </div>
                </div>
              </div>
              <div className="p-0">
                <SyntaxHighlighter
                  language="ruby"
                  style={theme === 'dark' ? oneDark : oneLight}
                  customStyle={{
                    margin: 0,
                    padding: '1rem',
                    background: 'transparent',
                    fontSize: '0.875rem',
                    lineHeight: '1.5',
                    borderRadius: '0 0 0.5rem 0.5rem',
                  }}
                  codeTagProps={{
                    style: {
                      fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
                    }
                  }}
                >
                  {selectedVersion.spec.code}
                </SyntaxHighlighter>
              </div>
            </div>
          ) : (
            <div className="text-center py-16">
              <Code className="h-16 w-16 text-gray-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2">No Code Available</h3>
              <p className="text-muted-foreground max-w-md mx-auto">
                {versions.length === 0
                  ? 'This agent has no synthesized versions yet. Code will appear here after the synthesis process completes successfully.'
                  : 'Select a version to view its code.'
                }
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Agent Version Details */}
      {selectedVersion && (
        <Card>
          <CardHeader>
            <CardTitle>Version Details</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <p className="text-sm font-medium text-muted-foreground">Version Name</p>
                <p className="text-sm font-mono">{selectedVersion.metadata.name}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Created</p>
                <p className="text-sm">{formatTimeAgo(selectedVersion.metadata.creationTimestamp)}</p>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Source Type</p>
                <Badge variant="outline">{selectedVersion.spec.sourceType || 'manual'}</Badge>
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Status</p>
                <Badge variant={selectedVersion.status?.phase === 'Ready' ? 'default' : 'secondary'}>
                  {selectedVersion.status?.phase || 'Unknown'}
                </Badge>
              </div>
            </div>

            {selectedVersion.spec.description && (
              <div className="mt-4">
                <p className="text-sm font-medium text-muted-foreground">Description</p>
                <p className="text-sm">{selectedVersion.spec.description}</p>
              </div>
            )}

            {selectedVersion.spec.optimizedTasks && Object.keys(selectedVersion.spec.optimizedTasks).length > 0 && (
              <div className="mt-4">
                <p className="text-sm font-medium text-muted-foreground mb-2">Optimized Tasks</p>
                <div className="space-y-2">
                  {Object.entries(selectedVersion.spec.optimizedTasks).map(([taskName, task]: [string, any]) => (
                    <div key={taskName} className="flex items-center justify-between p-2 bg-muted rounded">
                      <span className="text-sm font-medium">{task.name}</span>
                      <div className="flex items-center gap-2">
                        {task.confidenceScore !== undefined && (
                          <Badge variant="outline" className="text-xs">
                            {task.confidenceScore}% confidence
                          </Badge>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Synthesis Details */}
      <Card>
        <CardHeader>
          <CardTitle>Synthesis Details</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-3">
            <div>
              <p className="text-sm font-medium text-muted-foreground">Synthesis Status</p>
              <Badge variant={isSynthesized ? 'default' : 'secondary'}>
                {isSynthesized ? 'Code Synthesized' : 'Not Synthesized'}
              </Badge>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Last Synthesis</p>
              <p className="text-sm">
                {synthesisInfo?.lastSynthesisTime
                  ? formatTimeAgo(synthesisInfo.lastSynthesisTime)
                  : 'Never'
                }
              </p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Synthesis Model</p>
              <p className="text-sm">
                {synthesisInfo?.synthesisModel || 'N/A'}
              </p>
            </div>
          </div>

          {synthesisInfo && (
            <div className="mt-4 pt-4 border-t">
              <div className="grid gap-4 md:grid-cols-4 text-sm">
                <div>
                  <p className="font-medium text-muted-foreground">Duration</p>
                  <p>{synthesisInfo.synthesisDuration ? `${synthesisInfo.synthesisDuration.toFixed(2)}s` : 'N/A'}</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Attempts</p>
                  <p>{synthesisInfo.synthesisAttempts || 0}</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Code Hash</p>
                  <p className="font-mono text-xs">{synthesisInfo.codeHash?.substring(0, 12) || 'N/A'}...</p>
                </div>
                <div>
                  <p className="font-medium text-muted-foreground">Instructions Hash</p>
                  <p className="font-mono text-xs">{synthesisInfo.instructionsHash?.substring(0, 12) || 'N/A'}...</p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Rollback Confirmation Dialog */}
      <Dialog open={showRollbackDialog} onOpenChange={setShowRollbackDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Roll Back Agent Version</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <p className="text-sm text-muted-foreground">
              This will change the agent "{agent.metadata.name || 'unknown'}" to use version {selectedVersion?.spec?.version}
              instead of the current version. This action cannot be undone automatically.
            </p>
            <div className="flex items-center space-x-2">
              <input
                type="checkbox"
                id="lockOnRollback"
                checked={lockOnRollback}
                onChange={(e) => setLockOnRollback(e.target.checked)}
                className="rounded border-gray-300"
              />
              <label htmlFor="lockOnRollback" className="text-sm">
                Lock version after rollback (prevents automatic optimization)
              </label>
            </div>
          </div>
          <DialogFooter className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setShowRollbackDialog(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleRollback}
              disabled={rollbackMutation.isPending}
            >
              {rollbackMutation.isPending ? 'Rolling Back...' : 'Roll Back'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
