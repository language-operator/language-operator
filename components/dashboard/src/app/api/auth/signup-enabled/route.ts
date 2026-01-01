import { NextResponse } from 'next/server'
import { isSignupsDisabled } from '@/lib/env'

export async function GET() {
  return NextResponse.json({ 
    signupEnabled: !isSignupsDisabled 
  })
}