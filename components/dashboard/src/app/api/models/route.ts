import { NextResponse } from 'next/server'

// Base models route - for compatibility with TypeScript validator
export async function GET() {
  return NextResponse.json({ 
    message: "Use cluster-scoped endpoints: /api/clusters/[name]/models" 
  }, { status: 410 })
}