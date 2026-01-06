'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { usePlanInfo } from '@/hooks/useOrganizationUsage'
import { Check, Star } from 'lucide-react'
import { cn } from '@/lib/utils'

// Plan definition with quota information
const PLAN_DEFINITIONS = {
  free: {
    name: 'Pioneer',
    displayName: 'Pioneer',
    description: 'Perfect for exploration and learning',
    price: '$0/mo',
    compute: {
      'CPU Cores': '2 vCPU',
      'Memory': '4 GB'
    },
    limits: {
      'Clusters': '1',
      'Agents': '3',
      'Personas': '3',
      'Models': '2',
      'Tools': '5'
    },
    ctaText: 'Current Plan',
    ctaVariant: 'outline' as const,
    popular: false
  },
  pro: {
    name: 'Homesteader',
    displayName: 'Homesteader', 
    description: 'Scaled operations with advanced capabilities',
    price: '$49/month',
    compute: {
      'CPU Cores': '8 vCPU',
      'Memory': '16 GB'
    },
    limits: {
      'Clusters': '5',
      'Agents': '25',
      'Personas': '25',
      'Models': '10',
      'Tools': '50'
    },
    ctaText: 'Expand Your Horizon',
    ctaVariant: 'default' as const,
    popular: true
  },
  enterprise: {
    name: 'Landowner',
    displayName: 'Landowner',
    description: 'Unlimited scale for large operations', 
    price: 'Custom',
    compute: {
      'CPU Cores': 'Unlimited',
      'Memory': 'Unlimited'
    },
    limits: {
      'Clusters': 'Unlimited',
      'Agents': 'Unlimited',
      'Personas': 'Unlimited',
      'Models': 'Unlimited',
      'Tools': 'Unlimited'
    },
    ctaText: 'Claim The Territory',
    ctaVariant: 'outline' as const,
    popular: false
  }
}

interface PlanCardProps {
  planKey: keyof typeof PLAN_DEFINITIONS
  isCurrentPlan: boolean
}

function PlanCard({ planKey, isCurrentPlan }: PlanCardProps) {
  const plan = PLAN_DEFINITIONS[planKey]
  
  return (
    <Card className={cn(
      "relative h-full",
      isCurrentPlan && "ring-2 ring-amber-600 dark:ring-amber-400",
      plan.popular && !isCurrentPlan && "ring-2 ring-stone-900 dark:ring-stone-700"
    )}>
      {/* Popular Badge */}
      {plan.popular && (
        <div className="absolute -top-3 left-1/2 transform -translate-x-1/2">
          <Badge className="bg-amber-600 hover:bg-amber-700 text-white font-light tracking-wide px-3 py-1">
            <Star className="w-3 h-3 mr-1" />
            Standard
          </Badge>
        </div>
      )}
      
      {/* Current Plan Badge */}
      {isCurrentPlan && (
        <div className="absolute -top-3 left-1/2 transform -translate-x-1/2">
          <Badge variant="outline" className="bg-white dark:bg-stone-900 border-amber-600 text-amber-600 font-light tracking-wide px-3 py-1">
            Current Plan
          </Badge>
        </div>
      )}

      <CardHeader className="text-center space-y-4 pb-6">
        <div className="space-y-2">
          <CardTitle className="text-[13px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400 py-4">
            {plan.name}
          </CardTitle>
          <p className="text-sm font-light text-stone-600 dark:text-stone-400 leading-relaxed">
            {plan.description}
          </p>
        </div>
        
        <div className="space-y-1">
          <div className="text-2xl font-light text-stone-900 dark:text-stone-300">
            {plan.price}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {/* Compute Resources */}
        <div className="space-y-3">
          <h4 className="text-[11px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
            Compute
          </h4>
          <div className="space-y-2">
            {Object.entries(plan.compute).map(([resource, limit]) => (
              <div key={resource} className="flex justify-between items-center">
                <span className="text-xs font-light text-stone-700 dark:text-stone-300">
                  {resource}
                </span>
                <span className="text-xs font-light text-stone-900 dark:text-stone-200">
                  {limit}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Resource Limits */}
        <div className="space-y-3">
          <h4 className="text-[11px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
            Limits
          </h4>
          <div className="space-y-2">
            {Object.entries(plan.limits).map(([resource, limit]) => (
              <div key={resource} className="flex justify-between items-center">
                <span className="text-xs font-light text-stone-700 dark:text-stone-300">
                  {resource}
                </span>
                <span className="text-xs font-light text-stone-900 dark:text-stone-200">
                  {limit}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* CTA Button */}
        <div className="mt-auto pt-4">
          <Button 
            variant={isCurrentPlan ? "outline" : "default"}
            className={cn(
              "w-full font-light tracking-wide transition-colors",
              isCurrentPlan ? "opacity-75" : "opacity-75 bg-amber-600 hover:bg-amber-700 text-white"
            )}
            disabled={true}
          >
            <span className="text-sm">
              {isCurrentPlan ? 'Current Plan' : plan.ctaText}
            </span>
          </Button>
          
          <p className="text-[10px] font-light text-stone-500 dark:text-stone-500 text-center mt-2">
            Plan upgrades available soon
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

export function PlanComparison() {
  const { currentPlan } = usePlanInfo()

  return (
    <div className="space-y-8">
      {/* Section Header */}
      <div className="text-center space-y-2">
        <h3 className="text-[13px] tracking-widest uppercase font-light text-stone-600 dark:text-stone-400">
          Choose Your Plan
        </h3>
        <p className="text-sm font-light text-stone-600 dark:text-stone-400 max-w-2xl mx-auto leading-relaxed">
          Select the plan that matches your organization's scale and ambitions.
        </p>
      </div>

      {/* Plan Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {(Object.keys(PLAN_DEFINITIONS) as Array<keyof typeof PLAN_DEFINITIONS>).map((planKey) => (
          <PlanCard 
            key={planKey} 
            planKey={planKey} 
            isCurrentPlan={currentPlan === planKey}
          />
        ))}
      </div>
      
      {/* Footer Note */}
      <div className="text-center">
        <p className="text-xs font-light text-stone-500 dark:text-stone-500">
          Need something different? Contact us for custom enterprise solutions designed for your specific requirements.
        </p>
      </div>
    </div>
  )
}