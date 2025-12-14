import { NextResponse } from 'next/server'
import { fetchToolCatalog } from '@/lib/tool-catalog'

export async function GET() {
  try {
    const catalog = await fetchToolCatalog()
    
    return NextResponse.json(catalog)
  } catch (error) {
    console.error('Error fetching tool catalog:', error)
    
    return NextResponse.json(
      { 
        error: 'Failed to fetch tool catalog',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    )
  }
}