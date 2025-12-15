import { AlertCircle, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface NotFoundProps {
  title?: string
  message?: string
  onBack?: () => void
  backLabel?: string
}

export function NotFound({ 
  title = 'Not Found', 
  message = 'The resource you are looking for could not be found.',
  onBack,
  backLabel = 'Go Back'
}: NotFoundProps) {
  return (
    <div className="flex items-center justify-center h-64">
      <div className="text-center">
        <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
        <h3 className="text-lg font-medium mb-2">{title}</h3>
        <p className="text-muted-foreground mb-4">
          {message}
        </p>
        {onBack && (
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            {backLabel}
          </Button>
        )}
      </div>
    </div>
  )
}