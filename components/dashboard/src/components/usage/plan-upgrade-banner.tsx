'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { usePlanInfo } from '@/hooks/useOrganizationUsage'
import { ArrowRight, Zap, Users, Cpu, Building } from 'lucide-react'

const PLAN_FEATURES = {
  free: {
    title: 'Free Plan',
    subtitle: 'Perfect for exploring the frontier',
    features: ['3 agents', '2 models', '1 GB memory', '1 CPU core'],
    limitations: 'Limited resources for small experiments'
  },
  pro: {
    title: 'Professional Plan',
    subtitle: 'Built for the modern operator',
    features: ['25 agents', '10 models', '16 GB memory', '8 CPU cores'],
    limitations: 'Designed for growing AI operations'
  },
  enterprise: {
    title: 'Enterprise Plan',
    subtitle: 'Unlimited horizons await',
    features: ['Unlimited agents', 'Unlimited models', '64 GB memory', '32 CPU cores'],
    limitations: 'Full-scale AI deployment capabilities'
  },
  custom: {
    title: 'Custom Plan',
    subtitle: 'Tailored for your landscape',
    features: ['Custom resources', 'Flexible quotas', 'Dedicated support'],
    limitations: 'Configured specifically for your organization'
  }
}

const UPGRADE_PATHS = {
  free: {
    nextPlan: 'pro',
    nextTitle: 'Professional',
    callToAction: 'Expand Your Horizon',
    description: 'Scale your AI operations with expanded resources and advanced capabilities.'
  },
  pro: {
    nextPlan: 'enterprise',
    nextTitle: 'Enterprise',
    callToAction: 'Claim The Territory',
    description: 'Unlock unlimited potential with enterprise-grade resources and priority support.'
  }
}

function PlanFeatureIcon({ feature }: { feature: string }) {
  if (feature.includes('agent')) return <Users className="h-3 w-3" />
  if (feature.includes('model')) return <Zap className="h-3 w-3" />
  if (feature.includes('memory') || feature.includes('CPU')) return <Cpu className="h-3 w-3" />
  return <Building className="h-3 w-3" />
}

function CurrentPlanDisplay() {
  const { currentPlan, displayName, description } = usePlanInfo()
  const planInfo = PLAN_FEATURES[currentPlan as keyof typeof PLAN_FEATURES]

  return (
    <div className="space-y-6">
      {/* Current Plan Header */}
      <div className="space-y-2">
        <h3 className="text-[13px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
          Current Plan
        </h3>
        <div className="space-y-1">
          <h4 className="text-lg font-light text-stone-900 dark:text-stone-300">
            {planInfo.title}
          </h4>
          <p className="text-sm font-light text-stone-600 dark:text-stone-400">
            {planInfo.subtitle}
          </p>
        </div>
      </div>

      {/* Features List */}
      <div className="space-y-3">
        <h5 className="text-[11px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
          Included Resources
        </h5>
        <div className="grid grid-cols-2 gap-2">
          {planInfo.features.map((feature, index) => (
            <div key={index} className="flex items-center gap-2">
              <PlanFeatureIcon feature={feature} />
              <span className="text-xs font-light text-stone-700 dark:text-stone-300">
                {feature}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function UpgradePrompt() {
  const { currentPlan, isUpgradeable } = usePlanInfo()
  
  if (!isUpgradeable) {
    return (
      <div className="text-center space-y-4">
        <div className="space-y-2">
          <h4 className="text-sm font-light text-stone-900 dark:text-stone-300">
            You're on the highest tier
          </h4>
          <p className="text-xs font-light text-stone-600 dark:text-stone-400">
            Contact support for custom enterprise solutions
          </p>
        </div>
      </div>
    )
  }

  const upgradeInfo = UPGRADE_PATHS[currentPlan as keyof typeof UPGRADE_PATHS]
  
  if (!upgradeInfo) {
    return null
  }

  return (
    <div className="space-y-6">
      {/* Upgrade Header */}
      <div className="space-y-2">
        <h3 className="text-[13px] tracking-widest uppercase font-light text-amber-600 dark:text-amber-400">
          {upgradeInfo.callToAction}
        </h3>
        <div className="space-y-1">
          <h4 className="text-lg font-light text-stone-900 dark:text-stone-300">
            Upgrade to {upgradeInfo.nextTitle}
          </h4>
          <p className="text-sm font-light text-stone-600 dark:text-stone-400 leading-relaxed">
            {upgradeInfo.description}
          </p>
        </div>
      </div>

      {/* Upgrade Action */}
      <Button 
        className="w-full bg-amber-600 hover:bg-amber-700 text-white font-light tracking-wide transition-colors"
        disabled
      >
        <span className="text-sm">Contact Sales</span>
        <ArrowRight className="ml-2 h-4 w-4" />
      </Button>
      
      <p className="text-[10px] font-light text-stone-500 dark:text-stone-500 text-center">
        Plan upgrades will be available in a future release
      </p>
    </div>
  )
}

export function PlanUpgradeBanner() {
  return (
    <Card className="gap-0 py-0">
      <CardHeader className="px-8 py-6">
        <CardTitle className="text-sm font-light text-stone-600 dark:text-stone-400">
          Organization Plan
        </CardTitle>
      </CardHeader>
      
      <CardContent className="px-8 pb-8">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Current Plan Information */}
          <CurrentPlanDisplay />
          
          {/* Upgrade Section */}
          <div className="lg:border-l lg:border-stone-200 lg:dark:border-stone-800 lg:pl-8">
            <UpgradePrompt />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}