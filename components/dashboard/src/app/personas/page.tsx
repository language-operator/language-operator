import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Users, AlertCircle, CheckCircle, User, Briefcase, GraduationCap, Heart } from 'lucide-react'

export default function PersonasPage() {
  const personas = [
    {
      id: 'persona-1',
      name: 'helpful-assistant',
      description: 'Friendly and supportive assistant that provides helpful guidance',
      status: 'Active',
      category: 'General',
      icon: User,
      personality: 'Warm, empathetic, patient',
      tone: 'Professional but friendly',
      specialization: 'Customer support, general assistance',
      lastUsed: '5 minutes ago',
      agentCount: 8,
      namespace: 'production',
      traits: ['helpful', 'patient', 'clear communication', 'empathetic'],
    },
    {
      id: 'persona-2',
      name: 'technical-expert',
      description: 'Analytical and precise persona for technical tasks and code review',
      status: 'Active',
      category: 'Technical',
      icon: GraduationCap,
      personality: 'Analytical, detail-oriented, methodical',
      tone: 'Professional and precise',
      specialization: 'Code review, technical documentation, system analysis',
      lastUsed: '1 hour ago',
      agentCount: 3,
      namespace: 'development',
      traits: ['analytical', 'precise', 'methodical', 'logical'],
    },
    {
      id: 'persona-3',
      name: 'creative-writer',
      description: 'Imaginative and expressive persona for content creation',
      status: 'Active',
      category: 'Creative',
      icon: Heart,
      personality: 'Creative, expressive, engaging',
      tone: 'Conversational and inspiring',
      specialization: 'Blog posts, marketing copy, creative content',
      lastUsed: '3 hours ago',
      agentCount: 2,
      namespace: 'marketing',
      traits: ['creative', 'expressive', 'engaging', 'inspiring'],
    },
    {
      id: 'persona-4',
      name: 'business-analyst',
      description: 'Strategic and data-driven persona for business insights',
      status: 'Pending',
      category: 'Business',
      icon: Briefcase,
      personality: 'Strategic, data-driven, results-oriented',
      tone: 'Professional and authoritative',
      specialization: 'Market analysis, business strategy, reporting',
      lastUsed: '2 days ago',
      agentCount: 1,
      namespace: 'business',
      traits: ['strategic', 'analytical', 'results-oriented', 'data-driven'],
    },
    {
      id: 'persona-5',
      name: 'customer-advocate',
      description: 'Empathetic persona focused on customer satisfaction and resolution',
      status: 'Error',
      category: 'Customer Service',
      icon: Heart,
      personality: 'Empathetic, solution-focused, diplomatic',
      tone: 'Warm and understanding',
      specialization: 'Customer complaints, issue resolution, relationship building',
      lastUsed: '1 week ago',
      agentCount: 0,
      namespace: 'customer-service',
      traits: ['empathetic', 'diplomatic', 'solution-focused', 'understanding'],
    },
  ]

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'Active':
        return <CheckCircle className="h-4 w-4 text-green-500" />
      case 'Pending':
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
      case 'Error':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Active':
        return 'bg-green-100 text-green-800'
      case 'Pending':
        return 'bg-yellow-100 text-yellow-800'
      case 'Error':
        return 'bg-red-100 text-red-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getCategoryColor = (category: string) => {
    switch (category) {
      case 'General':
        return 'bg-blue-100 text-blue-800'
      case 'Technical':
        return 'bg-green-100 text-green-800'
      case 'Creative':
        return 'bg-purple-100 text-purple-800'
      case 'Business':
        return 'bg-orange-100 text-orange-800'
      case 'Customer Service':
        return 'bg-pink-100 text-pink-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Language Personas</h1>
            <p className="text-gray-600 mt-2">
              Define personality traits and communication styles for your agents
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            Create Persona
          </Button>
        </div>

        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Personas</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{personas.length}</div>
            </CardContent>
          </Card>
          
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active</CardTitle>
              <CheckCircle className="h-4 w-4 text-green-500" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {personas.filter(p => p.status === 'Active').length}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Categories</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {new Set(personas.map(p => p.category)).size}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Using Agents</CardTitle>
              <AlertCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {personas.reduce((sum, p) => sum + p.agentCount, 0)}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Personas List */}
        <div className="space-y-4">
          {personas.map((persona) => {
            const IconComponent = persona.icon
            return (
              <Card key={persona.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-4">
                      <IconComponent className="h-6 w-6 text-pink-500 mt-1" />
                      <div>
                        <div className="flex items-center space-x-2">
                          <CardTitle className="text-lg">{persona.name}</CardTitle>
                          <Badge className={getCategoryColor(persona.category)}>
                            {persona.category}
                          </Badge>
                          <Badge variant="secondary" className="text-xs">
                            {persona.namespace}
                          </Badge>
                        </div>
                        <CardDescription className="mt-1">
                          {persona.description}
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(persona.status)}
                      <Badge className={getStatusColor(persona.status)}>
                        {persona.status}
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Personality</h4>
                      <p className="text-sm">{persona.personality}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Tone</h4>
                      <p className="text-sm">{persona.tone}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Specialization</h4>
                      <p className="text-sm">{persona.specialization}</p>
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-600 mb-1">Agent Usage</h4>
                      <p className="text-sm">{persona.agentCount} agents</p>
                    </div>
                  </div>
                  
                  <div className="mt-4">
                    <h4 className="text-sm font-medium text-gray-600 mb-2">Traits</h4>
                    <div className="flex flex-wrap gap-1">
                      {persona.traits.map((trait) => (
                        <Badge key={trait} variant="outline" className="text-xs">
                          {trait}
                        </Badge>
                      ))}
                    </div>
                  </div>

                  <div className="flex space-x-2 mt-4">
                    <Button variant="outline" size="sm">
                      Edit
                    </Button>
                    <Button variant="outline" size="sm">
                      Clone
                    </Button>
                    <Button variant="outline" size="sm">
                      View Agents
                    </Button>
                    <Button variant="destructive" size="sm">
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}