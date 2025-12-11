import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Bot, Cpu, Wrench, Users, Cloud, Activity, TrendingUp, Clock } from 'lucide-react'

export default function Home() {
  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Dashboard Overview</h1>
          <p className="text-gray-600 mt-2">
            Monitor and manage your Language Operator resources
          </p>
        </div>

        {/* Stats Grid */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">12</div>
              <p className="text-xs text-muted-foreground">
                +2 from last week
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Models</CardTitle>
              <Cpu className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">4</div>
              <p className="text-xs text-muted-foreground">
                3 active, 1 pending
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Tools</CardTitle>
              <Wrench className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">8</div>
              <p className="text-xs text-muted-foreground">
                All operational
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Personas</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">6</div>
              <p className="text-xs text-muted-foreground">
                Across 2 categories
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Language Clusters</CardTitle>
              <Cloud className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">2</div>
              <p className="text-xs text-muted-foreground">
                Production & staging
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">System Health</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">99.2%</div>
              <p className="text-xs text-muted-foreground">
                30-day uptime
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Activity and Quick Actions */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5" />
                Recent Activity
              </CardTitle>
              <CardDescription>
                Latest changes to your resources
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex items-start space-x-3">
                  <Bot className="h-5 w-5 text-blue-500 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900">
                      Agent "customer-support-v2" created
                    </p>
                    <p className="text-sm text-gray-500">
                      2 hours ago in production namespace
                    </p>
                  </div>
                </div>
                
                <div className="flex items-start space-x-3">
                  <Cpu className="h-5 w-5 text-green-500 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900">
                      Model "gpt-4-turbo" configuration updated
                    </p>
                    <p className="text-sm text-gray-500">
                      4 hours ago
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3">
                  <Cloud className="h-5 w-5 text-orange-500 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900">
                      Cluster "production" scaled to 3 replicas
                    </p>
                    <p className="text-sm text-gray-500">
                      6 hours ago
                    </p>
                  </div>
                </div>

                <div className="flex items-start space-x-3">
                  <Wrench className="h-5 w-5 text-purple-500 mt-0.5" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900">
                      Tool "web-search" added to toolkit
                    </p>
                    <p className="text-sm text-gray-500">
                      1 day ago
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <TrendingUp className="h-5 w-5" />
                System Status
              </CardTitle>
              <CardDescription>
                Current health of your infrastructure
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">API Gateway</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Healthy</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">PostgreSQL Database</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Connected</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Language Clusters</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-yellow-500"></div>
                    <span className="text-sm text-yellow-600">Scaling</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Kubernetes API</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Available</span>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Monitoring</span>
                  <div className="flex items-center space-x-2">
                    <div className="h-2 w-2 rounded-full bg-green-500"></div>
                    <span className="text-sm text-green-600">Active</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
            <CardDescription>
              Get started with common tasks
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                <Bot className="h-8 w-8 text-blue-500 mb-2" />
                <span className="text-sm font-medium">Create Agent</span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  Build a new AI agent
                </span>
              </div>
              
              <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                <Cpu className="h-8 w-8 text-green-500 mb-2" />
                <span className="text-sm font-medium">Add Model</span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  Connect to LLM provider
                </span>
              </div>
              
              <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                <Wrench className="h-8 w-8 text-purple-500 mb-2" />
                <span className="text-sm font-medium">Configure Tool</span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  Add capabilities to agents
                </span>
              </div>
              
              <div className="flex flex-col items-center p-4 border rounded-lg hover:bg-gray-50 cursor-pointer transition-colors">
                <Cloud className="h-8 w-8 text-orange-500 mb-2" />
                <span className="text-sm font-medium">Deploy Cluster</span>
                <span className="text-xs text-gray-500 text-center mt-1">
                  Expose agents via HTTP
                </span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AuthenticatedLayout>
  )
}
