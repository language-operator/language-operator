'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
// import { Progress } from '@/components/ui/progress'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from '@/components/ui/dialog'
import { Brain, Zap, AlertTriangle, CheckCircle, Clock, Loader2 } from 'lucide-react'
import { LanguageAgent } from '@/types/agent'
import { useTriggerOptimization } from '@/hooks/use-agents'

interface OptimizationStatusProps {
  agent: LanguageAgent
  clusterName: string
}

const LEARNING_THRESHOLD = 10 // Default threshold from learning controller

export function OptimizationStatus({ agent, clusterName }: OptimizationStatusProps) {
  const [showOptimizeDialog, setShowOptimizeDialog] = useState(false)
  const optimizeMutation = useTriggerOptimization(clusterName)

  const runsPending = agent.status?.runsPendingLearning || 0
  const progressPercentage = Math.min((runsPending / LEARNING_THRESHOLD) * 100, 100)
  const isEligible = runsPending >= LEARNING_THRESHOLD

  // Check if agent version is locked
  const isLocked = agent.spec?.agentVersionRef?.lock || false

  const handleOptimize = async () => {
    try {
      await optimizeMutation.mutateAsync({
        agentName: agent.metadata?.name || '',
      })
      setShowOptimizeDialog(false)
    } catch (error) {
      console.error('Optimization failed:', error)
    }
  }

  const getStatusInfo = () => {
    if (isLocked) {
      return {
        status: 'locked',
        title: 'Version Locked',
        description: 'Agent version is locked and cannot be optimized automatically',
        icon: AlertTriangle,
        variant: 'secondary' as const,
        color: 'text-yellow-600'
      }
    }

    if (runsPending === 0) {
      return {
        status: 'no-data',
        title: 'No Optimization Data',
        description: 'No execution runs recorded yet',
        icon: Clock,
        variant: 'secondary' as const,
        color: 'text-gray-600'
      }
    }

    if (isEligible) {
      return {
        status: 'eligible',
        title: 'Eligible for Optimization',
        description: `Agent has ${runsPending} execution runs. Optimization can improve performance by learning from patterns.`,
        icon: Brain,
        variant: 'default' as const,
        color: 'text-green-600'
      }
    }

    return {
      status: 'accumulating',
      title: 'Accumulating Data',
      description: `${LEARNING_THRESHOLD - runsPending} more runs needed for optimization`,
      icon: Zap,
      variant: 'outline' as const,
      color: 'text-blue-600'
    }
  }

  const statusInfo = getStatusInfo()

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <statusInfo.icon className={`h-5 w-5 ${statusInfo.color}`} />
            Agent Optimization
          </CardTitle>
          <CardDescription>
            Automatic code optimization based on execution patterns
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Status Badge and Title */}
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <h3 className="font-medium">{statusInfo.title}</h3>
              <p className="text-sm text-muted-foreground">
                {statusInfo.description}
              </p>
            </div>
            <Badge variant={statusInfo.variant}>
              {statusInfo.status === 'eligible' && 'Ready'}
              {statusInfo.status === 'accumulating' && 'Learning'}
              {statusInfo.status === 'locked' && 'Locked'}
              {statusInfo.status === 'no-data' && 'Waiting'}
            </Badge>
          </div>

          {/* Progress Bar (only show if accumulating or eligible) */}
          {statusInfo.status === 'accumulating' || statusInfo.status === 'eligible' ? (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Execution runs collected</span>
                <span className="font-medium">{runsPending} / {LEARNING_THRESHOLD}</span>
              </div>
              {/* <Progress value={progressPercentage} className="h-2" /> */}
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-blue-600 h-2 rounded-full transition-all" 
                  style={{width: `${progressPercentage}%`}}
                ></div>
              </div>
              {statusInfo.status === 'eligible' && (
                <p className="text-xs text-green-600">
                  <CheckCircle className="inline h-3 w-3 mr-1" />
                  Threshold reached - ready for optimization
                </p>
              )}
            </div>
          ) : null}

          {/* Manual Optimization Button */}
          {isEligible && !isLocked && (
            <Button 
              onClick={() => setShowOptimizeDialog(true)}
              disabled={optimizeMutation.isPending}
              className="w-full"
            >
              {optimizeMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Optimizing...
                </>
              ) : (
                <>
                  <Brain className="h-4 w-4 mr-2" />
                  Optimize Agent
                </>
              )}
            </Button>
          )}

          {/* Information Alerts */}
          {isLocked && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                Agent version is locked. Unlock the version to enable automatic optimization.
              </AlertDescription>
            </Alert>
          )}

          {statusInfo.status === 'no-data' && (
            <Alert>
              <Clock className="h-4 w-4" />
              <AlertDescription>
                Start executing tasks with this agent to begin collecting optimization data.
              </AlertDescription>
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Optimization Confirmation Dialog */}
      <Dialog open={showOptimizeDialog} onOpenChange={setShowOptimizeDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Brain className="h-5 w-5" />
              Optimize Agent
            </DialogTitle>
            <DialogDescription>
              This will analyze execution patterns and generate optimized code for this agent.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <div className="grid gap-3">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Agent:</span>
                <span className="font-medium">{agent.metadata?.name}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Execution runs:</span>
                <span className="font-medium">{runsPending}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Optimization type:</span>
                <span className="font-medium">Pattern-based learning</span>
              </div>
            </div>
            <Alert>
              <Brain className="h-4 w-4" />
              <AlertDescription>
                The optimization process will analyze execution traces and create an improved 
                version of the agent code. This may take a few minutes to complete.
              </AlertDescription>
            </Alert>
          </div>
          <DialogFooter className="flex justify-end space-x-2">
            <Button
              variant="outline"
              onClick={() => setShowOptimizeDialog(false)}
              disabled={optimizeMutation.isPending}
            >
              Cancel
            </Button>
            <Button
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
                  Start Optimization
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}