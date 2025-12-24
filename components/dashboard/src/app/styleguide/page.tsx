'use client'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { useTheme } from 'next-themes'

export default function StyleGuidePage() {
  const { theme, setTheme } = useTheme()

  return (
    <div className="max-w-6xl mx-auto space-y-12">
      {/* Header */}
      <div className="space-y-4">
        <h1 className="text-[13px] tracking-widest uppercase font-light text-stone-900 dark:text-stone-300">
          Marfa Design System
        </h1>
        <p className="text-sm font-light text-stone-600 dark:text-stone-400">
          A minimalist design system inspired by Donald Judd and the West Texas landscape of Marfa.
        </p>
        
        {/* Theme Toggle */}
        <div className="flex gap-2">
          <Button 
            variant={theme === 'light' ? 'default' : 'outline'} 
            size="sm"
            onClick={() => setTheme('light')}
          >
            West Texas Day
          </Button>
          <Button 
            variant={theme === 'dark' ? 'default' : 'outline'} 
            size="sm"
            onClick={() => setTheme('dark')}
          >
            West Texas Night
          </Button>
        </div>
      </div>

      {/* Colors */}
      <Card className="gap-6 py-6">
        <CardHeader className="px-generous [.border-b]:pb-generous">
          <CardTitle>Color Palette</CardTitle>
          <CardDescription>Stone/amber for light mode, sage/fire for dark mode</CardDescription>
        </CardHeader>
        <CardContent className="px-generous space-y-6">
          <div>
            <h3 className="text-[11px] tracking-wider uppercase font-light mb-4">Light Mode: Stone & Amber</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="space-y-2">
                <div className="h-16 bg-stone-50 border border-stone-200"></div>
                <p className="text-[10px] tracking-widest uppercase">Stone-50</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-stone-600 border border-stone-200"></div>
                <p className="text-[10px] tracking-widest uppercase">Stone-600</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-stone-900 border border-stone-200"></div>
                <p className="text-[10px] tracking-widest uppercase">Stone-900</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-amber-900 border border-stone-200"></div>
                <p className="text-[10px] tracking-widest uppercase">Amber-900</p>
              </div>
            </div>
          </div>
          
          <div>
            <h3 className="text-[11px] tracking-wider uppercase font-light mb-4">Dark Mode: Sage & Fire</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="space-y-2">
                <div className="h-16 bg-stone-300 border border-stone-600"></div>
                <p className="text-[10px] tracking-widest uppercase">Stone-300 (Moonlight)</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-stone-400 border border-stone-600"></div>
                <p className="text-[10px] tracking-widest uppercase">Stone-400 (Sage)</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-amber-600 border border-stone-600"></div>
                <p className="text-[10px] tracking-widest uppercase">Amber-600 (Fire)</p>
              </div>
              <div className="space-y-2">
                <div className="h-16 bg-amber-400 border border-stone-600"></div>
                <p className="text-[10px] tracking-widest uppercase">Amber-400 (Starlight)</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Typography */}
      <Card className="gap-6 py-6">
        <CardHeader className="px-generous [.border-b]:pb-generous">
          <CardTitle>Typography</CardTitle>
          <CardDescription>System fonts with extended tracking and light weight</CardDescription>
        </CardHeader>
        <CardContent className="px-generous space-y-6">
          <div className="space-y-4">
            <div>
              <p className="text-[11px] tracking-wider uppercase font-light mb-2">Header Style</p>
              <h1 className="text-[13px] tracking-widest uppercase font-light text-stone-900 dark:text-stone-300">
                Language Operator
                <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse ml-1" />
              </h1>
            </div>
            
            <div>
              <p className="text-[11px] tracking-wider uppercase font-light mb-2">Label Style</p>
              <Label>Resource Name</Label>
            </div>
            
            <div>
              <p className="text-[11px] tracking-wider uppercase font-light mb-2">Button Text</p>
              <p className="text-[11px] tracking-wider uppercase font-light">Create Agent</p>
            </div>
            
            <div>
              <p className="text-[11px] tracking-wider uppercase font-light mb-2">Body Text</p>
              <p className="text-sm font-light text-stone-600 dark:text-stone-400">
                This is regular body text with light font weight for enhanced readability.
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Buttons */}
      <Card>
        <CardHeader>
          <CardTitle>Buttons</CardTitle>
          <CardDescription>Stone gradients with firelight hover states</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="space-y-2">
              <Button className="w-full">Default</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">Primary</p>
            </div>
            <div className="space-y-2">
              <Button variant="outline" className="w-full">Outline</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">Secondary</p>
            </div>
            <div className="space-y-2">
              <Button variant="ghost" className="w-full">Ghost</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">Minimal</p>
            </div>
            <div className="space-y-2">
              <Button variant="destructive" className="w-full">Delete</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">Destructive</p>
            </div>
          </div>
          
          <Separator className="my-6" />
          
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Button size="sm" className="w-full">Small</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">h-10</p>
            </div>
            <div className="space-y-2">
              <Button size="default" className="w-full">Default</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">h-12</p>
            </div>
            <div className="space-y-2">
              <Button size="lg" className="w-full">Large</Button>
              <p className="text-[10px] tracking-widest uppercase text-center">h-14</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Form Elements */}
      <Card>
        <CardHeader>
          <CardTitle>Form Elements</CardTitle>
          <CardDescription>Stone backgrounds with amber focus rings</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="email">Email Address</Label>
            <Input id="email" type="email" placeholder="Enter your email" />
          </div>
          
          <div className="space-y-2">
            <Label htmlFor="message">Message</Label>
            <Textarea id="message" placeholder="Enter your message" />
          </div>
          
          <div className="space-y-2">
            <Label>Status Badges</Label>
            <div className="flex gap-2">
              <Badge variant="default">Ready</Badge>
              <Badge variant="secondary">Pending</Badge>
              <Badge variant="destructive">Failed</Badge>
              <Badge variant="outline">Unknown</Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Cards */}
      <Card>
        <CardHeader>
          <CardTitle>Card Components</CardTitle>
          <CardDescription>Clean rectangular containers with warm/night shadows</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Card className="gap-6 py-6">
              <CardHeader className="px-generous [.border-b]:pb-generous">
                <CardTitle>Basic Card</CardTitle>
                <CardDescription>Simple card with header and content</CardDescription>
              </CardHeader>
              <CardContent className="px-generous">
                <p className="text-sm font-light">
                  Card content with generous padding and clean typography.
                </p>
              </CardContent>
            </Card>
            
            <Card className="gap-6 py-6">
              <CardHeader className="px-generous [.border-b]:pb-generous">
                <CardTitle>Status Card</CardTitle>
                <CardDescription>Card with status information</CardDescription>
              </CardHeader>
              <CardContent className="px-generous space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-light">Status</span>
                  <Badge>Active</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm font-light">Count</span>
                  <span className="text-sm font-light">42</span>
                </div>
              </CardContent>
            </Card>
          </div>
        </CardContent>
      </Card>

      {/* Tables */}
      <Card>
        <CardHeader>
          <CardTitle>Table Component</CardTitle>
          <CardDescription>Data tables with stone borders and typography</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell className="font-light">my-cluster</TableCell>
                <TableCell><Badge variant="default">Ready</Badge></TableCell>
                <TableCell className="font-light">2 hours ago</TableCell>
                <TableCell>
                  <Button variant="ghost" size="sm">Edit</Button>
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-light">test-cluster</TableCell>
                <TableCell><Badge variant="secondary">Pending</Badge></TableCell>
                <TableCell className="font-light">1 day ago</TableCell>
                <TableCell>
                  <Button variant="ghost" size="sm">Edit</Button>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Tabs Navigation */}
      <Card>
        <CardHeader>
          <CardTitle>Tab Navigation</CardTitle>
          <CardDescription>Subnav pattern with amber underlines and icon support - the standard for resource detail pages</CardDescription>
        </CardHeader>
        <CardContent className="space-y-8">
          
          {/* Basic Tabs */}
          <div className="space-y-4">
            <h3 className="text-[11px] tracking-wider uppercase font-light text-stone-600 dark:text-stone-400">Basic Text Tabs</h3>
            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList>
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="metrics">Metrics</TabsTrigger>
                <TabsTrigger value="logs">Logs</TabsTrigger>
                <TabsTrigger value="yaml">YAML</TabsTrigger>
              </TabsList>
              <TabsContent value="overview">
                <div className="space-y-4">
                  <h4 className="text-[11px] tracking-wider uppercase font-light">Overview Content</h4>
                  <p className="text-sm font-light">
                    This is the overview tab content with Marfa design system styling.
                  </p>
                </div>
              </TabsContent>
              <TabsContent value="metrics">
                <div className="space-y-4">
                  <h4 className="text-[11px] tracking-wider uppercase font-light">Metrics Content</h4>
                  <p className="text-sm font-light">
                    Metrics and analytics content would be displayed here.
                  </p>
                </div>
              </TabsContent>
              <TabsContent value="logs">
                <div className="space-y-4">
                  <h4 className="text-[11px] tracking-wider uppercase font-light">Logs Content</h4>
                  <p className="text-sm font-light">
                    Log output with terminal-style formatting.
                  </p>
                </div>
              </TabsContent>
              <TabsContent value="yaml">
                <div className="space-y-4">
                  <h4 className="text-[11px] tracking-wider uppercase font-light">YAML Content</h4>
                  <p className="text-sm font-light">
                    Raw YAML configuration display.
                  </p>
                </div>
              </TabsContent>
            </Tabs>
          </div>

          {/* Subnav with Icons */}
          <div className="space-y-4">
            <h3 className="text-[11px] tracking-wider uppercase font-light text-stone-600 dark:text-stone-400">Resource Detail Subnav (Recommended)</h3>
            <p className="text-xs font-light text-stone-600 dark:text-stone-400">
              Standard pattern for agent, model, tool, and persona detail pages. Features icons, clean typography, and amber active states.
            </p>
            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList>
                <TabsTrigger value="overview">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 5a2 2 0 012-2h4a2 2 0 012 2v14l-5-3-5 3V5z" />
                  </svg>
                  Overview
                </TabsTrigger>
                <TabsTrigger value="details">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Details
                </TabsTrigger>
                <TabsTrigger value="metrics">
                  <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                  </svg>
                  Metrics
                </TabsTrigger>
              </TabsList>
              <TabsContent value="overview">
                <Card className="p-6">
                  <h4 className="text-[11px] tracking-wider uppercase font-light mb-4">Resource Overview</h4>
                  <p className="text-sm font-light">
                    Primary resource information including status, configuration, and key metrics. 
                    This is the default view for most resource detail pages.
                  </p>
                </Card>
              </TabsContent>
              <TabsContent value="details">
                <Card className="p-6">
                  <h4 className="text-[11px] tracking-wider uppercase font-light mb-4">Detailed Configuration</h4>
                  <p className="text-sm font-light">
                    Complete configuration details, advanced settings, and technical specifications.
                    Used for deep inspection of resource properties.
                  </p>
                </Card>
              </TabsContent>
              <TabsContent value="metrics">
                <Card className="p-6">
                  <h4 className="text-[11px] tracking-wider uppercase font-light mb-4">Performance Metrics</h4>
                  <p className="text-sm font-light">
                    Real-time and historical performance data, charts, and monitoring information.
                    Essential for operational insights and debugging.
                  </p>
                </Card>
              </TabsContent>
            </Tabs>
          </div>

          {/* Implementation Code Example */}
          <div className="space-y-4">
            <h3 className="text-[11px] tracking-wider uppercase font-light text-stone-600 dark:text-stone-400">Implementation</h3>
            <div className="bg-stone-50 border border-stone-200 p-4 dark:bg-stone-800/50 dark:border-stone-700">
              <pre className="text-xs font-mono text-stone-900 dark:text-stone-300 overflow-x-auto">
{`<Tabs value={activeTab} onValueChange={setActiveTab}>
  <TabsList>
    <TabsTrigger value="overview">
      <Home className="w-4 h-4 mr-2" />
      Overview
    </TabsTrigger>
    <TabsTrigger value="details">
      <Info className="w-4 h-4 mr-2" />
      Details
    </TabsTrigger>
    <TabsTrigger value="metrics">
      <BarChart3 className="w-4 h-4 mr-2" />
      Metrics
    </TabsTrigger>
  </TabsList>

  <TabsContent value="overview">
    <ResourceOverview />
  </TabsContent>
  
  <TabsContent value="details">
    <ResourceDetails />
  </TabsContent>
  
  <TabsContent value="metrics">
    <ResourceMetrics />
  </TabsContent>
</Tabs>`}
              </pre>
            </div>
          </div>

          {/* Usage Guidelines */}
          <div className="space-y-4">
            <h3 className="text-[11px] tracking-wider uppercase font-light text-stone-600 dark:text-stone-400">Usage Guidelines</h3>
            <div className="grid md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <h4 className="text-xs font-light text-green-700 dark:text-green-400">✓ Do</h4>
                <ul className="text-xs font-light text-stone-600 dark:text-stone-400 space-y-1">
                  <li>• Use icons for resource detail pages</li>
                  <li>• Keep tab labels concise and clear</li>
                  <li>• Follow Overview → Details → Metrics pattern</li>
                  <li>• Use consistent icons across similar resources</li>
                  <li>• Maintain logical information hierarchy</li>
                </ul>
              </div>
              <div className="space-y-2">
                <h4 className="text-xs font-light text-red-700 dark:text-red-400">✗ Don't</h4>
                <ul className="text-xs font-light text-stone-600 dark:text-stone-400 space-y-1">
                  <li>• Use more than 5 tabs in a single nav</li>
                  <li>• Mix icons and non-icon tabs in same set</li>
                  <li>• Create deeply nested tab structures</li>
                  <li>• Use tabs for primary navigation</li>
                  <li>• Override the Marfa tab styling</li>
                </ul>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Dialog */}
      <Card>
        <CardHeader>
          <CardTitle>Dialog Component</CardTitle>
          <CardDescription>Modal dialogs with backdrop blur</CardDescription>
        </CardHeader>
        <CardContent>
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline">Open Dialog</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Example Dialog</DialogTitle>
                <DialogDescription>
                  This dialog demonstrates the Marfa design system styling with proper backdrop blur and stone colors.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Name</Label>
                  <Input id="name" placeholder="Enter resource name" />
                </div>
                <div className="flex gap-2">
                  <Button>Save Changes</Button>
                  <Button variant="outline">Cancel</Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </CardContent>
      </Card>

      {/* Spacing */}
      <Card>
        <CardHeader>
          <CardTitle>Spacing System</CardTitle>
          <CardDescription>Generous padding and comfortable gaps</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-[11px] tracking-wider uppercase font-light mb-2">Base Unit: 12px</p>
            <div className="h-3 bg-stone-200 dark:bg-stone-700"></div>
          </div>
          
          <div>
            <p className="text-[11px] tracking-wider uppercase font-light mb-2">Comfortable: 24px</p>
            <div className="h-6 bg-stone-200 dark:bg-stone-700"></div>
          </div>
          
          <div>
            <p className="text-[11px] tracking-wider uppercase font-light mb-2">Generous: 48px</p>
            <div className="h-12 bg-stone-200 dark:bg-stone-700"></div>
          </div>
        </CardContent>
      </Card>

      {/* Design Principles */}
      <Card>
        <CardHeader>
          <CardTitle>Design Principles</CardTitle>
          <CardDescription>West Texas minimalism philosophy</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="text-[11px] tracking-wider uppercase font-light mb-2">Light Mode ☀️</h3>
              <ul className="space-y-1 text-sm font-light text-stone-600">
                <li>• Embrace white space and desert light</li>
                <li>• Stone as primary, amber as warmth</li>
                <li>• Warm brown shadows (not black)</li>
                <li>• Restrained earth tones</li>
              </ul>
            </div>
            
            <div>
              <h3 className="text-[11px] tracking-wider uppercase font-light mb-2">Dark Mode 🌌</h3>
              <ul className="space-y-1 text-sm font-light text-stone-400">
                <li>• Embrace vast darkness and starlight</li>
                <li>• Deep blacks with stone undertones</li>
                <li>• Sage moonlight for readable text</li>
                <li>• Fire colors revealed through interaction</li>
              </ul>
            </div>
          </div>
          
          <Separator />
          
          <div>
            <h3 className="text-[11px] tracking-wider uppercase font-light mb-2">Universal Principles</h3>
            <ul className="space-y-1 text-sm font-light">
              <li>• Pure geometric precision (no rounded corners)</li>
              <li>• Typography as sculptural element (extended tracking)</li>
              <li>• Material honesty (no decoration)</li>
              <li>• Warmth revealed through interaction</li>
              <li>• Maximum negative space is intentional</li>
              <li>• Light font weight (300) only</li>
            </ul>
          </div>
        </CardContent>
      </Card>

      {/* Footer */}
      <div className="text-center space-y-2">
        <p className="text-[11px] font-light text-stone-600 dark:text-stone-400">
          Marfa Design System — Inspired by Donald Judd & West Texas
        </p>
        <p className="text-[10px] tracking-widest uppercase font-light text-stone-500">
          Not just a design system — an aesthetic philosophy
        </p>
      </div>
    </div>
  )
}