export interface FileEntry {
  name: string
  path: string
  type: 'file' | 'directory'
  size?: number
  modified?: string
  permissions?: string
  isHidden?: boolean
}

export interface WorkspaceState {
  files: FileEntry[]
  currentPath: string
  loading: boolean
  error?: string
}

export interface WorkspaceUploadRequest {
  path: string
  filename: string
  content: string // base64 encoded
  size: number
}

export interface WorkspaceFileContent {
  content: string
  encoding: 'utf8' | 'binary' | 'base64'
  size: number
  mimeType?: string
}

export interface WorkspacePodInfo {
  name: string
  namespace: string
  status: 'pending' | 'running' | 'failed' | 'succeeded'
  createdAt: string
  expiresAt: string
}

export class WorkspaceError extends Error {
  code: 'NOT_FOUND' | 'ACCESS_DENIED' | 'SIZE_LIMIT' | 'INVALID_PATH' | 'POD_ERROR' | 'TIMEOUT'
  details?: Record<string, any>

  constructor(code: WorkspaceError['code'], message: string, details?: Record<string, any>) {
    super(message)
    this.name = 'WorkspaceError'
    this.code = code
    this.details = details
  }
}

// File size limits
export const WORKSPACE_LIMITS = {
  MAX_VIEW_SIZE: 10 * 1024 * 1024, // 10MB for viewing
  MAX_UPLOAD_SIZE: 100 * 1024 * 1024, // 100MB for upload
  MAX_FILES_PER_DIRECTORY: 1000, // Prevent UI overload
  POD_TIMEOUT_MINUTES: 30, // Auto-expire file manager pods
} as const

// File types that can be viewed in the browser
export const VIEWABLE_FILE_EXTENSIONS = [
  '.txt', '.md', '.json', '.yaml', '.yml', '.xml', '.html', '.css', '.js', '.ts',
  '.jsx', '.tsx', '.py', '.rb', '.go', '.rs', '.java', '.c', '.cpp', '.h', '.hpp',
  '.sh', '.bash', '.zsh', '.fish', '.sql', '.toml', '.ini', '.conf', '.cfg',
  '.log', '.csv', '.env', '.gitignore', '.dockerignore', '.makefile', '.dockerfile',
  '.bas', '.basic', '.qb', '.vb',
] as const

// Files to hide from workspace browser for security
export const HIDDEN_FILE_PATTERNS = [
  /^\.git$/,
  /^\.env$/,
  /^\.env\..+$/,
  /.*\.key$/,
  /.*\.pem$/,
  /.*\.p12$/,
  /.*\.keystore$/,
  /^\.ssh$/,
  /^\.aws$/,
  /^\.kube$/,
] as const

export type ViewableFileExtension = typeof VIEWABLE_FILE_EXTENSIONS[number]