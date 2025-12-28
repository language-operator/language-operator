'use client'

import { useEffect, useState } from 'react'
import { LanguageAgent } from '@/types/agent'
import { FileEntry } from '@/types/workspace'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  ChevronRight, 
  ChevronDown, 
  Folder, 
  FolderOpen, 
  File, 
  FileText, 
  FileCode,
  Image,
  Archive,
  AlertCircle
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useApiClient } from '@/lib/api-client'

interface WorkspaceFileTreeProps {
  agent: LanguageAgent
  clusterName: string
  currentPath: string
  onFileSelect: (file: FileEntry) => void
  onPathChange: (path: string) => void
  refreshTrigger: number
}

interface DirectoryNode extends FileEntry {
  children?: DirectoryNode[]
  isExpanded?: boolean
  isLoading?: boolean
}

export function WorkspaceFileTree({
  agent,
  clusterName,
  currentPath,
  onFileSelect,
  onPathChange,
  refreshTrigger,
}: WorkspaceFileTreeProps) {
  const [tree, setTree] = useState<DirectoryNode[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const apiClient = useApiClient()

  const agentName = agent.metadata?.name

  const fetchDirectory = async (path: string): Promise<FileEntry[]> => {
    const response = await apiClient.get(
      `/clusters/${clusterName}/agents/${agentName}/workspace/files?path=${encodeURIComponent(path)}`
    )
    
    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(errorData.error || 'Failed to fetch directory')
    }
    
    const data = await response.json()
    return data.files || []
  }

  const loadRootDirectory = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const files = await fetchDirectory('/')
      const rootNodes: DirectoryNode[] = files.map(file => ({
        ...file,
        children: file.type === 'directory' ? [] : undefined,
        isExpanded: false,
        isLoading: false,
      }))
      setTree(rootNodes)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const toggleDirectory = async (node: DirectoryNode) => {
    if (node.type !== 'directory') return

    const updateTree = (nodes: DirectoryNode[]): DirectoryNode[] => {
      return nodes.map(n => {
        if (n.path === node.path) {
          if (n.isExpanded) {
            // Collapse
            return { ...n, isExpanded: false }
          } else {
            // Expand
            return { ...n, isExpanded: true, isLoading: true }
          }
        }
        if (n.children) {
          return { ...n, children: updateTree(n.children) }
        }
        return n
      })
    }

    setTree(updateTree(tree))

    // Load children if expanding
    if (!node.isExpanded) {
      try {
        const children = await fetchDirectory(node.path)
        const childNodes: DirectoryNode[] = children.map(file => ({
          ...file,
          children: file.type === 'directory' ? [] : undefined,
          isExpanded: false,
          isLoading: false,
        }))

        setTree(nodes => {
          const updateWithChildren = (nodes: DirectoryNode[]): DirectoryNode[] => {
            return nodes.map(n => {
              if (n.path === node.path) {
                return { ...n, children: childNodes, isLoading: false }
              }
              if (n.children) {
                return { ...n, children: updateWithChildren(n.children) }
              }
              return n
            })
          }
          return updateWithChildren(nodes)
        })
      } catch (err: any) {
        // Handle error loading children
        setTree(nodes => {
          const updateWithError = (nodes: DirectoryNode[]): DirectoryNode[] => {
            return nodes.map(n => {
              if (n.path === node.path) {
                return { ...n, isExpanded: false, isLoading: false }
              }
              if (n.children) {
                return { ...n, children: updateWithError(n.children) }
              }
              return n
            })
          }
          return updateWithError(nodes)
        })
      }
    }
  }

  const handleNodeClick = (node: DirectoryNode) => {
    if (node.type === 'directory') {
      toggleDirectory(node)
      onPathChange(node.path)
    } else {
      onFileSelect(node)
    }
  }

  const getFileIcon = (file: DirectoryNode) => {
    if (file.type === 'directory') {
      return file.isExpanded ? (
        <FolderOpen className="w-4 h-4 text-blue-500" />
      ) : (
        <Folder className="w-4 h-4 text-blue-500" />
      )
    }

    const extension = file.name.toLowerCase().split('.').pop()
    
    // Code files
    if (['js', 'ts', 'jsx', 'tsx', 'py', 'rb', 'go', 'rs', 'java', 'c', 'cpp', 'h'].includes(extension || '')) {
      return <FileCode className="w-4 h-4 text-green-500" />
    }
    
    // Text files
    if (['txt', 'md', 'json', 'yaml', 'yml', 'xml', 'csv', 'log'].includes(extension || '')) {
      return <FileText className="w-4 h-4 text-gray-500" />
    }
    
    // Image files
    if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp'].includes(extension || '')) {
      return <Image className="w-4 h-4 text-purple-500" />
    }
    
    // Archive files
    if (['zip', 'tar', 'gz', 'rar', '7z'].includes(extension || '')) {
      return <Archive className="w-4 h-4 text-orange-500" />
    }
    
    // Default file icon
    return <File className="w-4 h-4 text-gray-400" />
  }

  const renderNode = (node: DirectoryNode, depth = 0) => {
    return (
      <div key={node.path}>
        <Button
          variant="ghost"
          className={cn(
            "w-full justify-start h-8 px-2 py-1 hover:bg-muted/60",
            currentPath === node.path && "bg-muted"
          )}
          style={{ paddingLeft: `${depth * 12 + 8}px` }}
          onClick={() => handleNodeClick(node)}
        >
          <div className="flex items-center space-x-2 min-w-0 flex-1">
            {node.type === 'directory' && (
              node.isLoading ? (
                <div className="w-4 h-4" />
              ) : node.isExpanded ? (
                <ChevronDown className="w-4 h-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="w-4 h-4 text-muted-foreground" />
              )
            )}
            {getFileIcon(node)}
            <span className="truncate text-sm" style={{ textTransform: 'none' }}>{node.name}</span>
            {node.type === 'file' && node.size !== undefined && (
              <span className="text-xs text-muted-foreground ml-auto">
                {formatFileSize(node.size)}
              </span>
            )}
          </div>
        </Button>
        
        {node.isExpanded && node.children && (
          <div>
            {node.children.map(child => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    )
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  useEffect(() => {
    if (agentName) {
      loadRootDirectory()
    }
  }, [agentName, refreshTrigger])

  if (loading) {
    return (
      <div className="space-y-2">
        <div className="text-sm font-medium mb-4">Files</div>
        {[...Array(6)].map((_, i) => (
          <Skeleton key={i} className="h-8 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    const isPodError = error.includes('Pod failed to start') || error.includes('Pod status check failed')
    
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <AlertCircle className="w-8 h-8 text-destructive mb-2" />
        <p className="text-sm font-medium">
          {isPodError ? 'Workspace pod needs restart' : 'Error loading workspace'}
        </p>
        <p className="text-xs text-muted-foreground mb-4">
          {isPodError ? 
            'The workspace container has expired and needs to be restarted. Click "Restart Workspace" to create a new container.' :
            error
          }
        </p>
        <Button variant="outline" size="sm" onClick={loadRootDirectory}>
          {isPodError ? 'Restart Workspace' : 'Try Again'}
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-0 p-4">
      <div className="text-sm font-medium mb-4 flex items-center">
        <FolderOpen className="w-4 h-4 mr-2" />
        Files
      </div>
      <div className="h-[calc(100vh-300px)] overflow-y-auto">
        {tree.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Folder className="w-8 h-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">Empty workspace</p>
          </div>
        ) : (
          tree.map(node => renderNode(node))
        )}
      </div>
    </div>
  )
}