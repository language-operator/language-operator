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

      {/* Tabs */}
      <Card>
        <CardHeader>
          <CardTitle>Tabs Component</CardTitle>
          <CardDescription>Navigation tabs with border-based active states</CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="overview" className="space-y-4">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="metrics">Metrics</TabsTrigger>
              <TabsTrigger value="logs">Logs</TabsTrigger>
              <TabsTrigger value="yaml">YAML</TabsTrigger>
            </TabsList>
            <TabsContent value="overview">
              <div className="space-y-4">
                <h3 className="text-[11px] tracking-wider uppercase font-light">Overview Content</h3>
                <p className="text-sm font-light">
                  This is the overview tab content with Marfa design system styling.
                </p>
              </div>
            </TabsContent>
            <TabsContent value="metrics">
              <div className="space-y-4">
                <h3 className="text-[11px] tracking-wider uppercase font-light">Metrics Content</h3>
                <p className="text-sm font-light">
                  Metrics and analytics content would be displayed here.
                </p>
              </div>
            </TabsContent>
            <TabsContent value="logs">
              <div className="space-y-4">
                <h3 className="text-[11px] tracking-wider uppercase font-light">Logs Content</h3>
                <p className="text-sm font-light">
                  Log output with terminal-style formatting.
                </p>
              </div>
            </TabsContent>
            <TabsContent value="yaml">
              <div className="space-y-4">
                <h3 className="text-[11px] tracking-wider uppercase font-light">YAML Content</h3>
                <p className="text-sm font-light">
                  Raw YAML configuration display.
                </p>
              </div>
            </TabsContent>
          </Tabs>
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