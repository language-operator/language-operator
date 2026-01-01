import { NextRequest, NextResponse } from 'next/server'
import { isSignupsDisabled } from '@/lib/env'
import { isSignupAllowed } from '@/lib/invitation-utils'

export async function POST(req: NextRequest) {
  try {
    const { callbackUrl } = await req.json()

    const allowed = await isSignupAllowed(isSignupsDisabled, callbackUrl)
    
    const response = {
      allowed,
      message: allowed 
        ? 'Signup is allowed' 
        : 'Signup is currently disabled. Please use an invitation link to create an account.'
    }

    return NextResponse.json(response, { status: 200 })
  } catch (error: unknown) {
    console.error('Error checking signup permissions:', error)
    return NextResponse.json(
      { 
        allowed: false, 
        message: 'Unable to verify signup permissions.' 
      },
      { status: 500 }
    )
  }
}