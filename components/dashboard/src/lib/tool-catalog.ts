import yaml from 'js-yaml'
import { ToolCatalog, ToolCatalogEntry } from '@/types/tool-catalog'

const CATALOG_URL = 'https://raw.githubusercontent.com/language-operator/language-tools/main/index.yaml'
const CACHE_TTL = 5 * 60 * 1000 // 5 minutes

interface CacheEntry {
  data: ToolCatalog
  timestamp: number
}

let cache: CacheEntry | null = null

export async function fetchToolCatalog(): Promise<ToolCatalog> {
  // Check cache
  if (cache && Date.now() - cache.timestamp < CACHE_TTL) {
    return cache.data
  }

  try {
    const response = await fetch(CATALOG_URL, {
      headers: {
        'Accept': 'text/yaml,text/plain,*/*',
      },
      cache: 'no-cache',
    })

    if (!response.ok) {
      throw new Error(`Failed to fetch tool catalog: ${response.status} ${response.statusText}`)
    }

    const text = await response.text()
    const data = yaml.load(text) as ToolCatalog

    // Validate the structure
    if (!data.version || !data.tools || typeof data.tools !== 'object') {
      throw new Error('Invalid tool catalog format')
    }

    // Cache the result
    cache = {
      data,
      timestamp: Date.now(),
    }

    return data
  } catch (error) {
    console.error('Error fetching tool catalog:', error)
    
    // Return empty catalog on error
    return {
      version: '1.0',
      generated: new Date().toISOString(),
      tools: {},
    }
  }
}

export function getToolById(catalog: ToolCatalog, toolId: string): ToolCatalogEntry | undefined {
  return catalog.tools[toolId]
}

export function searchTools(catalog: ToolCatalog, query: string): ToolCatalogEntry[] {
  const lowercaseQuery = query.toLowerCase()
  
  return Object.values(catalog.tools).filter(tool => 
    tool.name.toLowerCase().includes(lowercaseQuery) ||
    tool.displayName.toLowerCase().includes(lowercaseQuery) ||
    tool.description.toLowerCase().includes(lowercaseQuery) ||
    tool.tags?.some(tag => tag.toLowerCase().includes(lowercaseQuery))
  )
}

export function getToolsByTag(catalog: ToolCatalog, tag: string): ToolCatalogEntry[] {
  return Object.values(catalog.tools).filter(tool => 
    tool.tags?.includes(tag)
  )
}

export function transformCatalogEntryToLanguageTool(
  entry: ToolCatalogEntry,
  namespace: string,
  clusterName?: string
): any {
  const languageTool = {
    apiVersion: 'langop.io/v1alpha1',
    kind: 'LanguageTool',
    metadata: {
      name: entry.name,
      namespace: namespace,
      labels: {
        'langop.io/source': 'catalog',
        'langop.io/catalog-name': entry.name,
      },
    },
    spec: {
      // Core LanguageTool spec fields
      type: entry.type,
      image: entry.image,
      deploymentMode: entry.deploymentMode,
      ...(entry.port && { port: entry.port }),
      ...(clusterName && { clusterRef: clusterName }),
      ...(entry.egress && { egress: entry.egress }),
    } as any,
  }

  return languageTool
}