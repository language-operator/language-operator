import {
  Boxes,
  Bot,
  Cpu,
  Wrench,
  Users,
  Zap,
  Building,
  MemoryStick,
  Server,
  type LucideProps
} from 'lucide-react'

// Domain-specific icon components for Language Operator resources

export function LanguageClusterIcon(props: LucideProps) {
  return <Boxes {...props} />
}

export function LanguageAgentIcon(props: LucideProps) {
  return <Bot {...props} />
}

export function LanguageModelIcon(props: LucideProps) {
  return <Cpu {...props} />
}

export function LanguageToolIcon(props: LucideProps) {
  return <Wrench {...props} />
}

export function LanguagePersonaIcon(props: LucideProps) {
  return <Users {...props} />
}

export function ComputeIcon(props: LucideProps) {
  return <Cpu {...props} />
}

export function MemoryIcon(props: LucideProps) {
  return <MemoryStick {...props} />
}

export function CapacityIcon(props: LucideProps) {
  return <Server {...props} />
}

// Plan and feature icons
export function PlanFeatureIcon({ feature, ...props }: { feature: string } & LucideProps) {
  if (feature.toLowerCase().includes('cluster')) return <LanguageClusterIcon {...props} />
  if (feature.toLowerCase().includes('agent')) return <LanguageAgentIcon {...props} />
  if (feature.toLowerCase().includes('model')) return <LanguageModelIcon {...props} />
  if (feature.toLowerCase().includes('tool')) return <LanguageToolIcon {...props} />
  if (feature.toLowerCase().includes('persona')) return <LanguagePersonaIcon {...props} />
  if (feature.toLowerCase().includes('memory')) return <MemoryIcon {...props} />
  if (feature.toLowerCase().includes('cpu') || feature.toLowerCase().includes('compute')) return <ComputeIcon {...props} />
  // Default fallback
  return <Building {...props} />
}

// Resource type helper
export function getResourceIcon(resourceType: string, props: LucideProps) {
  switch (resourceType.toLowerCase()) {
    case 'clusters':
      return <LanguageClusterIcon {...props} />
    case 'agents':
      return <LanguageAgentIcon {...props} />
    case 'models':
      return <LanguageModelIcon {...props} />
    case 'tools':
      return <LanguageToolIcon {...props} />
    case 'personas':
      return <LanguagePersonaIcon {...props} />
    case 'cpu':
    case 'compute':
      return <ComputeIcon {...props} />
    case 'memory':
      return <MemoryIcon {...props} />
    default:
      return <Building {...props} />
  }
}