'use client'

import { useState, useEffect, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { AlertCircle, Code, History, Lock, Unlock, RotateCcw, Brain, Loader2, Sparkles, Grid3X3, FileText, ChevronDown, ChevronRight, Trash2 } from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { useAgentVersions, useRollbackAgent, useToggleAgentLock, useTriggerOptimization, useDeleteAgentVersion } from '@/hooks/use-agents'
import { LanguageAgent } from '@/types/agent'
import { formatTimeAgo } from './utils'
import { toast } from 'sonner'

interface AgentCodeProps {
  agent: LanguageAgent
  clusterName: string
}

export function AgentCode({ agent, clusterName }: AgentCodeProps) {
  const [selectedVersionName, setSelectedVersionName] = useState<string>('')
  const [showRollbackDialog, setShowRollbackDialog] = useState(false)
  const [lockOnRollback, setLockOnRollback] = useState(false)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [versionToDelete, setVersionToDelete] = useState<string>('')
  const [viewMode, setViewMode] = useState<'raw' | 'graphical'>('graphical')
  const [showRaw, setShowRaw] = useState(false)
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set())
  const [isMainExpanded, setIsMainExpanded] = useState(false)
  const { theme } = useTheme()

  // Hooks for version management
  const { data: versionsResponse, isLoading: versionsLoading, error: versionsError } = useAgentVersions(agent.metadata.name || '', clusterName || '')
  const rollbackMutation = useRollbackAgent(clusterName || '')
  const lockMutation = useToggleAgentLock(clusterName || '')
  const optimizeMutation = useTriggerOptimization(clusterName || '')
  const deleteMutation = useDeleteAgentVersion(clusterName || '')

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
      // Success case
      toast.success("Optimization Started", {
        description: "Agent optimization has been triggered successfully."
      })
    } catch (error) {
      // Error cases with user-friendly messages
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      
      if (errorMessage.includes('Agent locked')) {
        toast.error("Agent Locked", {
          description: "Unlock the agent version before trying to optimize."
        })
      } else if (errorMessage.includes('Agent not found')) {
        toast.error("Agent Not Found", {
          description: "The specified agent could not be found."
        })
      } else {
        toast.error("Optimization Failed", {
          description: errorMessage
        })
      }
      
      // Still log to console for debugging
      console.error('Optimization failed:', error)
    }
  }

  const handleDeleteVersion = async () => {
    if (!versionToDelete) return

    try {
      const result = await deleteMutation.mutateAsync({
        agentName: agent.metadata.name || '',
        versionName: versionToDelete
      })
      setShowDeleteDialog(false)
      setVersionToDelete('')
      
      // Show success toast
      toast.success("Version Deleted", {
        description: result.message || `Version deleted successfully${result.rolledBackTo ? ' and agent rolled back' : ''}`
      })
    } catch (error) {
      console.error('Delete failed:', error)
      const errorMessage = error instanceof Error ? error.message : 'Unknown error'
      
      toast.error("Delete Failed", {
        description: errorMessage
      })
    }
  }

  const handleShowDeleteDialog = (versionName: string) => {
    setVersionToDelete(versionName)
    setShowDeleteDialog(true)
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

  // Parse Ruby DSL code using server-side AST parsing
  const [parsedTasks, setParsedTasks] = useState<Array<{
    id: string
    name: string
    type: 'symbolic' | 'neural'
    instructions: string
    inputs: Array<{name: string, type: string}>
    outputs: Array<{name: string, type: string}>
    isOptimized: boolean
    codeBlock?: string
  }>>([])
  const [mainBlock, setMainBlock] = useState<string | null>(null)
  
  const [parsingTasks, setParsingTasks] = useState(false)

  // Toggle task expansion
  const toggleTaskExpansion = (taskId: string) => {
    setExpandedTasks(prev => {
      const newSet = new Set(prev)
      if (newSet.has(taskId)) {
        newSet.delete(taskId)
      } else {
        newSet.add(taskId)
      }
      return newSet
    })
  }

  // Parse tasks when selected version changes
  useEffect(() => {
    const parseCode = async () => {
      if (!selectedVersion?.spec?.code) {
        setParsedTasks([])
        return
      }

      setParsingTasks(true)
      try {
        const response = await fetch('/api/parse-ruby', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ code: selectedVersion.spec.code })
        })

        const result = await response.json()
        
        if (result.success) {
          // Add optimization info and proper IDs
          const optimizedTasks = selectedVersion?.spec?.optimizedTasks || {}
          const tasksWithOptimization = result.data.tasks.map((task: any, index: number) => ({
            ...task,
            id: `task_${index + 1}`,
            isOptimized: task.name in optimizedTasks
          }))
          setParsedTasks(tasksWithOptimization)
          setMainBlock(result.data.mainBlock)
        } else {
          console.error('Failed to parse Ruby code:', result.error)
          setParsedTasks([])
          setMainBlock(null)
        }
      } catch (error) {
        console.error('Error calling Ruby parser API:', error)
        setParsedTasks([])
      } finally {
        setParsingTasks(false)
      }
    }

    parseCode()
  }, [selectedVersion?.spec?.code, selectedVersion?.spec?.optimizedTasks])

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
                          <Badge variant="secondary" className="text-xs flex items-center gap-1">
                            CURRENT
                            {isLocked && <Lock className="h-3 w-3" />}
                          </Badge>
                        )}
                        <Badge variant="outline" className="text-xs">
                          {version.spec.sourceType === 'learning' ? (
                            <div className="flex items-center gap-1">
                              <Sparkles className="h-3 w-3" />
                              Optimized
                            </div>
                          ) : (
                            version.spec.sourceType || 'manual'
                          )}
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

              {/* Delete Button - Only show for versions > v1 and enabled */}
              {selectedVersionName && selectedVersion?.spec?.version > 1 && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleShowDeleteDialog(selectedVersionName)}
                  disabled={
                    deleteMutation.isPending || 
                    (selectedVersion?.isCurrent && isLocked)
                  }
                  className="text-red-600 hover:text-red-700 hover:bg-red-50"
                >
                  {deleteMutation.isPending ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    <>
                      <Trash2 className="h-4 w-4 mr-2" />
                      Delete
                    </>
                  )}
                </Button>
              )}

              {/* Optimize Button - Only show for current manual versions */}
              {selectedVersion?.isCurrent && selectedVersion?.spec?.sourceType === 'manual' && (
                <Button
                  variant="default"
                  size="sm"
                  onClick={handleOptimize}
                  disabled={optimizeMutation.isPending || agent.status?.learningRequestPending}
                >
                  {optimizeMutation.isPending ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Optimizing...
                    </>
                  ) : agent.status?.learningRequestPending ? (
                    <>
                      <Brain className="h-4 w-4 mr-2" />
                      Optimization Pending...
                    </>
                  ) : (
                    <>
                      <Brain className="h-4 w-4 mr-2" />
                      Optimize
                    </>
                  )}
                </Button>
              )}

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

        </CardContent>
      </Card>

      {/* Agent Code with View Toggle */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Code className="h-5 w-5" />
                Agent Code {selectedVersion?.isCurrent ? '(Current)' : '(Version ' + selectedVersion?.spec?.version + ')'}
              </CardTitle>
              <CardDescription>
                Tasks and execution logic synthesized from agent instructions
              </CardDescription>
            </div>
            {/* Raw Toggle Switch */}
            <div className="flex items-center gap-3">
              <label htmlFor="raw-toggle" className="text-sm font-medium">
                Raw
              </label>
              <Switch
                id="raw-toggle"
                checked={showRaw}
                onCheckedChange={setShowRaw}
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {versionsError ? (
            <div className="text-center py-16">
              <AlertCircle className="h-16 w-16 text-red-400 mx-auto mb-4" />
              <h3 className="text-lg font-semibold mb-2 text-red-600">Error Loading Versions</h3>
              <p className="text-muted-foreground max-w-md mx-auto">{versionsError.message}</p>
            </div>
          ) : selectedVersion?.spec?.code ? (
            <div>

              {/* Raw View */}
              {showRaw ? (
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
                /* Graphical View */
                parsingTasks ? (
                  <div className="flex items-center justify-center py-16">
                    <div className="text-center">
                      <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
                      <p className="text-gray-600">Parsing agent tasks...</p>
                    </div>
                  </div>
                ) : parsedTasks.length > 0 ? (
                  <div className="space-y-6">
                    {/* Agent Summary */}
                    <div className="bg-muted p-4 rounded-lg">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                          <div>
                            <Badge variant="outline">
                              {selectedVersion.spec.sourceType === 'learning' ? 'Neural Agent' : 'Symbolic Agent'}
                            </Badge>
                          </div>
                          <div className="text-sm text-muted-foreground">
                            {parsedTasks.length} tasks ({parsedTasks.filter(t => t.isOptimized).length} optimized)
                          </div>
                        </div>
                        <Badge variant="outline">v{selectedVersion.spec.version}</Badge>
                      </div>
                    </div>

                    {/* Tasks Column */}
                    <div className="space-y-4">
                      {parsedTasks.map((task, index) => {
                        const isExpanded = expandedTasks.has(task.id)
                        return (
                          <Card key={task.id} className="transition-all duration-200">
                            <CardHeader 
                              className="py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                              onClick={() => toggleTaskExpansion(task.id)}
                            >
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                  <div className="bg-muted rounded-full w-8 h-8 flex items-center justify-center text-sm font-medium">
                                    {task.type === 'neural' ? (
                                      <Brain className="h-4 w-4" />
                                    ) : (
                                      <Code className="h-4 w-4" />
                                    )}
                                  </div>
                                  <CardTitle className="text-lg">{task.name}</CardTitle>
                                </div>
                                <div className="flex items-center gap-2">
                                <Badge variant={task.type === 'neural' ? 'default' : 'secondary'}>
                                  {task.type === 'neural' ? (
                                    <>
                                      <Brain className="h-3 w-3 mr-1" />
                                      Neural
                                    </>
                                  ) : (
                                    <>
                                      <Code className="h-3 w-3 mr-1" />
                                      Symbolic
                                    </>
                                  )}
                                </Badge>
                                  {task.isOptimized && (
                                    <span className="inline-flex items-center justify-center border px-3 py-1 text-[10px] tracking-wider uppercase font-light w-fit whitespace-nowrap shrink-0 gap-1 transition-colors overflow-hidden bg-amber-600 text-white border-amber-600 hover:bg-amber-700">
                                      <Sparkles className="size-3 pointer-events-none" />
                                      Optimized
                                    </span>
                                  )}
                                  {isExpanded ? (
                                    <ChevronDown className="h-4 w-4 text-muted-foreground" />
                                  ) : (
                                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                                  )}
                                </div>
                              </div>
                            </CardHeader>
                            {isExpanded && (
                              <CardContent className="space-y-4">
                            {/* Instructions */}
                            {task.instructions && (
                              <div>
                                <p className="text-sm font-medium text-muted-foreground mb-2">Instructions</p>
                                <p className="text-sm bg-muted p-3 rounded italic">"{task.instructions}"</p>
                              </div>
                            )}
                            
                            {/* Inputs/Outputs */}
                            {(task.inputs.length > 0 || task.outputs.length > 0) && (
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {task.inputs.length > 0 && (
                                  <div>
                                    <p className="text-sm font-medium text-muted-foreground mb-2">Inputs</p>
                                    <div className="flex flex-wrap gap-1">
                                      {task.inputs.map((input, idx) => (
                                        <Badge key={idx} variant="outline" className="text-xs">
                                          <strong>{input.name}</strong> {input.type}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                                
                                {task.outputs.length > 0 && (
                                  <div>
                                    <p className="text-sm font-medium text-muted-foreground mb-2">Outputs</p>
                                    <div className="flex flex-wrap gap-1">
                                      {task.outputs.map((output, idx) => (
                                        <Badge key={idx} variant="outline" className="text-xs">
                                          <strong>{output.name}</strong> {output.type}
                                        </Badge>
                                      ))}
                                    </div>
                                  </div>
                                )}
                              </div>
                            )}

                            {/* Code Block for Symbolic Tasks */}
                            {task.type === 'symbolic' && task.codeBlock && (
                              <div>
                                <p className="text-sm font-medium text-muted-foreground mb-2">Implementation</p>
                                <div className="border rounded-lg">
                                  <div className="bg-muted p-2 border-b">
                                    <p className="text-xs font-medium">Ruby</p>
                                  </div>
                                  <div className="p-0">
                                    <SyntaxHighlighter
                                      language="ruby"
                                      style={theme === 'dark' ? oneDark : oneLight}
                                      customStyle={{
                                        margin: 0,
                                        padding: '1rem',
                                        background: 'transparent',
                                        fontSize: '0.8rem',
                                        lineHeight: '1.4',
                                        borderRadius: '0 0 0.5rem 0.5rem',
                                      }}
                                      codeTagProps={{
                                        style: {
                                          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Monaco, Consolas, "Liberation Mono", "Courier New", monospace'
                                        }
                                      }}
                                    >
                                      {task.codeBlock}
                                    </SyntaxHighlighter>
                                  </div>
                                </div>
                              </div>
                            )}

                            {/* Neural Task Placeholder */}
                            {task.type === 'neural' && (
                              <div>
                                <p className="text-sm font-medium text-muted-foreground mb-2">Neural Implementation</p>
                                <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950 dark:to-purple-950 p-4 rounded-lg border border-dashed">
                                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                    <Brain className="h-4 w-4" />
                                    This task uses neural processing - no code implementation shown
                                  </div>
                                </div>
                              </div>
                              )}
                            </CardContent>
                            )}
                          </Card>
                        )
                      })}
                    </div>

                    {/* Main Block */}
                    {mainBlock && (
                      <Card className="transition-all duration-200">
                        <CardHeader 
                          className="py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                          onClick={() => setIsMainExpanded(!isMainExpanded)}
                        >
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                              <div className="bg-muted rounded-full w-8 h-8 flex items-center justify-center text-sm font-medium">
                                <RotateCcw className="h-4 w-4" />
                              </div>
                              <CardTitle className="text-lg">MAIN</CardTitle>
                            </div>
                            <div className="flex items-center gap-2">
                              <Badge variant="outline">
                                <RotateCcw className="h-3 w-3 mr-1" />
                                Workflow
                              </Badge>
                              {isMainExpanded ? (
                                <ChevronDown className="h-4 w-4 text-muted-foreground" />
                              ) : (
                                <ChevronRight className="h-4 w-4 text-muted-foreground" />
                              )}
                            </div>
                          </div>
                        </CardHeader>
                        {isMainExpanded && (
                          <CardContent>
                            <div className="border rounded-lg">
                              <div className="bg-muted p-3 border-b">
                                <p className="font-medium text-sm">Ruby</p>
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
                                  {mainBlock}
                                </SyntaxHighlighter>
                              </div>
                            </div>
                          </CardContent>
                        )}
                      </Card>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-16">
                    <Grid3X3 className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                    <h3 className="text-lg font-semibold mb-2">No Tasks Found</h3>
                    <p className="text-muted-foreground max-w-md mx-auto">
                      No tasks could be parsed from the agent code. Tasks may not be structured in the expected Ruby DSL format.
                    </p>
                  </div>
                )
              )}
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

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Agent Version</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <p className="text-sm text-muted-foreground">
              Are you sure you want to delete version {versions.find((v: any) => v.metadata.name === versionToDelete)?.spec?.version}? 
              This action cannot be undone.
            </p>
            {versionToDelete === currentVersionName && (
              <div className="p-3 bg-amber-50 border border-amber-200 rounded-md">
                <p className="text-sm text-amber-800">
                  <strong>Note:</strong> You are deleting the current version. The agent will automatically 
                  roll back to the previous version.
                </p>
              </div>
            )}
            {versionToDelete === currentVersionName && isLocked && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-md">
                <p className="text-sm text-red-800">
                  <strong>Cannot delete:</strong> This version is currently locked. Unlock it first to delete.
                </p>
              </div>
            )}
          </div>
          <DialogFooter className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setShowDeleteDialog(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteVersion}
              disabled={
                deleteMutation.isPending || 
                (versionToDelete === currentVersionName && isLocked)
              }
            >
              {deleteMutation.isPending ? 'Deleting...' : 'Delete Version'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
