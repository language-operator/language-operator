'use client'

import { useEffect, useState } from 'react'
import { LanguageAgent } from '@/types/agent'
import { FileEntry } from '@/types/workspace'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  Download, 
  Trash2, 
  Copy, 
  Check, 
  FileText, 
  File, 
  AlertCircle,
  Eye,
  EyeOff,
  ChevronLeft
} from 'lucide-react'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneLight, oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useTheme } from 'next-themes'
import { cn } from '@/lib/utils'
import { useToast } from '@/hooks/use-toast'
import { 
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

interface WorkspaceFileViewerProps {
  agent: LanguageAgent
  clusterName: string
  selectedFile: FileEntry | null
  onFileDelete: () => void
  onBack: () => void
}

interface FileContent {
  content: string
  encoding: string
  size: number
  filename: string
  path: string
  language: string
  mimeType: string
  isViewable: boolean
}

export function WorkspaceFileViewer({
  agent,
  clusterName,
  selectedFile,
  onFileDelete,
  onBack,
}: WorkspaceFileViewerProps) {
  const [fileContent, setFileContent] = useState<FileContent | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { theme } = useTheme()
  const [copied, setCopied] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const { toast } = useToast()

  const agentName = agent.metadata?.name

  const loadFileContent = async (file: FileEntry) => {
    if (!agentName) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(
        `/api/clusters/${clusterName}/agents/${agentName}/workspace/files/view?path=${encodeURIComponent(file.path)}`
      )
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to load file content')
      }
      
      const content = await response.json()
      setFileContent(content)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleDownload = async () => {
    if (!selectedFile || !agentName) return
    
    try {
      const response = await fetch(
        `/api/clusters/${clusterName}/agents/${agentName}/workspace/files/download?path=${encodeURIComponent(selectedFile.path)}`
      )
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Download failed')
      }
      
      // Create download link
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = selectedFile.name
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
      
      toast({
        title: 'Download started',
        description: `Downloading ${selectedFile.name}`,
      })
    } catch (err: any) {
      toast({
        title: 'Download failed',
        description: err.message,
        variant: 'destructive',
      })
    }
  }

  const handleCopyContent = async () => {
    if (!fileContent) return
    
    try {
      await navigator.clipboard.writeText(fileContent.content)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
      
      toast({
        title: 'Copied to clipboard',
        description: 'File content has been copied',
      })
    } catch (err) {
      toast({
        title: 'Copy failed',
        description: 'Failed to copy content to clipboard',
        variant: 'destructive',
      })
    }
  }

  const handleDelete = async () => {
    if (!selectedFile || !agentName) return
    
    setIsDeleting(true)
    
    try {
      const response = await fetch(
        `/api/clusters/${clusterName}/agents/${agentName}/workspace/files?path=${encodeURIComponent(selectedFile.path)}`,
        { method: 'DELETE' }
      )
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Delete failed')
      }
      
      toast({
        title: 'File deleted',
        description: `${selectedFile.name} has been deleted`,
      })
      
      onFileDelete()
    } catch (err: any) {
      toast({
        title: 'Delete failed',
        description: err.message,
        variant: 'destructive',
      })
    } finally {
      setIsDeleting(false)
    }
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  const formatDate = (dateStr?: string): string => {
    if (!dateStr) return 'Unknown'
    try {
      return new Date(dateStr).toLocaleString()
    } catch {
      return dateStr
    }
  }

  useEffect(() => {
    if (selectedFile && selectedFile.type === 'file') {
      loadFileContent(selectedFile)
    } else {
      setFileContent(null)
      setError(null)
    }
  }, [selectedFile])

  // Handle keyboard navigation - Escape key to go back
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onBack()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onBack])

  if (!selectedFile) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-8">
        <FileText className="w-16 h-16 text-muted-foreground mb-4" />
        <h3 className="text-lg font-medium mb-2">No file selected</h3>
        <p className="text-muted-foreground">
          Select a file from the file tree to view its contents
        </p>
      </div>
    )
  }

  if (selectedFile.type === 'directory') {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center p-8">
        <File className="w-16 h-16 text-muted-foreground mb-4" />
        <h3 className="text-lg font-medium mb-2">Directory selected</h3>
        <p className="text-muted-foreground">
          Directories cannot be viewed. Select a file to see its contents.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center space-x-4 min-w-0 flex-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={onBack}
            className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground shrink-0"
            title="Back to Files (Press Escape)"
            aria-label="Back to file list"
          >
            <ChevronLeft className="w-4 h-4" />
          </Button>
          <div className="min-w-0 flex-1">
            <div className="flex items-center space-x-2">
              <h3 className="font-medium truncate">{selectedFile.name}</h3>
            </div>
            <div className="flex items-center space-x-4 mt-1">
              <Badge variant="secondary" className="text-xs">
                {formatFileSize(selectedFile.size || 0)}
              </Badge>
              {selectedFile.modified && (
                <span className="text-xs text-muted-foreground">
                  Modified: {formatDate(selectedFile.modified)}
                </span>
              )}
              {fileContent && (
                <Badge variant="outline" className="text-xs">
                  {fileContent.language}
                </Badge>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center space-x-1">
          {fileContent && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCopyContent}
              className="h-8 w-8 p-0"
              title={copied ? 'Copied!' : 'Copy'}
            >
              {copied ? (
                <Check className="w-4 h-4" />
              ) : (
                <Copy className="w-4 h-4" />
              )}
            </Button>
          )}
          
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDownload}
            className="h-8 w-8 p-0"
            title="Download"
          >
            <Download className="w-4 h-4" />
          </Button>
          
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                disabled={isDeleting}
                title="Delete"
              >
                <Trash2 className="w-4 h-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete File</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete "{selectedFile.name}"? 
                  This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDelete}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                  disabled={isDeleting}
                >
                  {isDeleting ? 'Deleting...' : 'Delete'}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {loading ? (
          <div className="p-4 space-y-3">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-4/5" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full text-center p-8">
            <AlertCircle className="w-16 h-16 text-destructive mb-4" />
            <h3 className="text-lg font-medium mb-2">Error loading file</h3>
            <p className="text-muted-foreground mb-4">{error}</p>
            <div className="flex items-center space-x-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => loadFileContent(selectedFile)}
              >
                Try Again
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={handleDownload}
              >
                <Download className="w-4 h-4 mr-2" />
                Download
              </Button>
            </div>
          </div>
        ) : fileContent ? (
          <div className="h-full overflow-y-auto">
            <div className="p-4">
              {fileContent.language === 'text' || fileContent.language === 'markdown' ? (
                <pre className="whitespace-pre-wrap font-mono text-sm leading-relaxed">
                  {fileContent.content}
                </pre>
              ) : (
                <SyntaxHighlighter
                  language={fileContent.language}
                  style={theme === 'dark' ? oneDark : oneLight}
                  customStyle={{
                    margin: 0,
                    padding: 0,
                    background: 'transparent',
                  }}
                  wrapLines
                  wrapLongLines
                  showLineNumbers
                >
                  {fileContent.content}
                </SyntaxHighlighter>
              )}
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-full text-center p-8">
            <EyeOff className="w-16 h-16 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium mb-2">Cannot preview file</h3>
            <p className="text-muted-foreground mb-4">
              This file type is not supported for preview
            </p>
            <Button variant="outline" onClick={handleDownload}>
              <Download className="w-4 h-4 mr-2" />
              Download to view
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}