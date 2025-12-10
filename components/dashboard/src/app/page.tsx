import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'

export default function Home() {
  return (
    <AuthenticatedLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Overview</h1>
          <p className="text-gray-600 mt-2">
            Welcome to Language Operator Dashboard
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <div className="rounded-lg border bg-white p-6">
            <h3 className="text-sm font-medium text-gray-600">Total Agents</h3>
            <p className="text-3xl font-bold mt-2">0</p>
          </div>
          <div className="rounded-lg border bg-white p-6">
            <h3 className="text-sm font-medium text-gray-600">Total Models</h3>
            <p className="text-3xl font-bold mt-2">0</p>
          </div>
          <div className="rounded-lg border bg-white p-6">
            <h3 className="text-sm font-medium text-gray-600">Total Tools</h3>
            <p className="text-3xl font-bold mt-2">0</p>
          </div>
        </div>

        <div className="rounded-lg border bg-white p-6">
          <h2 className="text-xl font-semibold mb-4">Getting Started</h2>
          <p className="text-gray-600 mb-4">
            Your Language Operator dashboard is ready! Start by creating your first resources:
          </p>
          <ul className="list-disc list-inside space-y-2 text-gray-600">
            <li>Create a Language Model to connect to your LLM provider</li>
            <li>Define Language Tools for your agents to use</li>
            <li>Build Language Agents to automate tasks</li>
            <li>Configure Language Personas to customize agent behavior</li>
            <li>Set up Language Clusters to expose agents via HTTP</li>
          </ul>
        </div>
      </div>
    </AuthenticatedLayout>
  )
}
