'use client'

import { useParams } from 'next/navigation'
import { useModel } from '@/hooks/use-models'
import { Card, CardContent } from '@/components/ui/card'
import { BarChart3 } from 'lucide-react'
import { LanguageModel } from '@/types/model'

interface ModelMetricsProps {
  model: LanguageModel
}

function ModelMetrics({ model }: ModelMetricsProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[400px] space-y-4">
      <div className="mx-auto w-24 h-24 bg-stone-100 border border-stone-200 flex items-center justify-center dark:bg-stone-800 dark:border-stone-700">
        <BarChart3 className="h-12 w-12 text-stone-500 dark:text-stone-400" />
      </div>
      <h3 className="text-xl font-semibold text-stone-600 dark:text-stone-400">Coming Soon</h3>
    </div>
  )
}

export default function ModelMetricsPage() {
  const params = useParams()
  const clusterName = params.name as string
  const modelName = params.modelName as string

  const { data: modelResponse, isLoading } = useModel(modelName, clusterName)
  const model = modelResponse?.data

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardContent className="flex items-center justify-center py-16">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 mx-auto mb-4"></div>
              <p className="text-gray-600">Loading model metrics...</p>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!model) {
    return null // Layout handles error state
  }

  return <ModelMetrics model={model} />
}