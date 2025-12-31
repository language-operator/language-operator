import { NextRequest, NextResponse } from 'next/server'
import { checkAndPerformInitialSetup } from '@/lib/initial-setup'

export async function GET(request: NextRequest) {
  try {
    console.log('🔧 [DEBUG] Manual initial setup trigger called')
    const result = await checkAndPerformInitialSetup()
    
    return NextResponse.json({
      success: true,
      setupPerformed: result,
      message: result ? 'Initial setup completed' : 'Initial setup not needed or already done'
    })
  } catch (error) {
    console.error('🚨 [DEBUG] Initial setup failed:', error)
    return NextResponse.json({
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error'
    }, { status: 500 })
  }
}