import { toast } from 'sonner'

export function useToast() {
  return {
    toast: (options: {
      title?: string
      description?: string
      variant?: 'default' | 'destructive' | 'success'
    }) => {
      const { title, description, variant = 'default' } = options
      
      const message = title ? `${title}: ${description || ''}` : description || ''
      
      switch (variant) {
        case 'success':
          toast.success(message)
          break
        case 'destructive':
          toast.error(message)
          break
        default:
          toast(message)
          break
      }
    },
  }
}