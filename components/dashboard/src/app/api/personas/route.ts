import { NextResponse } from 'next/server'

// Base personas route - for compatibility with TypeScript validator
export async function GET() {
  return NextResponse.json({ 
    message: "Use cluster-scoped endpoints: /api/clusters/[name]/personas" 
  }, { status: 410 })
}