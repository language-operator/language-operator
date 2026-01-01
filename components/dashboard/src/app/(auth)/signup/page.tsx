'use client'

import { signIn } from 'next-auth/react'
import { useRouter, useSearchParams } from 'next/navigation'
import { useState, useEffect, Suspense } from 'react'

function SignupForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [signupAllowed, setSignupAllowed] = useState<boolean | null>(null)

  // Check if signup is allowed and pre-fill email from invitation if provided
  useEffect(() => {
    const checkSignupPermissions = async () => {
      const emailParam = searchParams.get('email')
      const callbackUrl = searchParams.get('callbackUrl')
      
      if (emailParam) {
        setEmail(emailParam)
      }

      try {
        const response = await fetch('/api/auth/signup/check', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ callbackUrl }),
        })
        
        const data = await response.json()
        setSignupAllowed(data.allowed)
        
        if (!data.allowed) {
          setError(data.message || 'Signup is currently disabled. Please contact an administrator for an invitation.')
        }
      } catch {
        setSignupAllowed(false)
        setError('Unable to verify signup permissions.')
      }
    }

    checkSignupPermissions()
  }, [searchParams, router])

  const handleSignup = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!signupAllowed) {
      setError('Signup is not allowed.')
      return
    }
    
    setIsLoading(true)
    setError('')

    try {
      const callbackUrl = searchParams.get('callbackUrl')
      const response = await fetch('/api/auth/signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email, password, callbackUrl }),
      })

      const data = await response.json()

      if (!response.ok) {
        setError(data.error || 'Failed to create account')
        return
      }

      // Auto sign in after successful signup
      const result = await signIn('credentials', {
        email,
        password,
        redirect: false,
      })

      if (result?.error) {
        setError('Account created but login failed. Please try signing in.')
      } else {
        const callbackUrl = searchParams.get('callbackUrl') || '/'
        router.push(callbackUrl)
        router.refresh()
      }
    } catch {
      setError('An error occurred. Please try again.')
    } finally {
      setIsLoading(false)
    }
  }

  // Show loading state while checking permissions
  if (signupAllowed === null) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 flex items-center justify-center p-8">
        <div className="w-full max-w-[480px]">
          <div className="bg-white/95 backdrop-blur-sm border border-stone-800/90 shadow-[0_8px_32px_rgba(120,53,15,0.08)] dark:bg-stone-900/95 dark:border-stone-700/90 dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]">
            <div className="border-b border-stone-800/80 dark:border-stone-600/80 p-12">
              <h1 className="text-[13px] font-light tracking-[0.2em] uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
                Language Operator
                <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
              </h1>
            </div>
            <div className="p-12 flex justify-center">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-stone-900 dark:border-amber-400"></div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // Redirect to login if signup is not allowed
  if (signupAllowed === false) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 flex items-center justify-center p-8">
        <div className="w-full max-w-[480px]">
          <div className="bg-white/95 backdrop-blur-sm border border-stone-800/90 shadow-[0_8px_32px_rgba(120,53,15,0.08)] dark:bg-stone-900/95 dark:border-stone-700/90 dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]">
            <div className="border-b border-stone-800/80 dark:border-stone-600/80 p-12">
              <h1 className="text-[13px] font-light tracking-[0.2em] uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
                Language Operator
                <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
              </h1>
            </div>
            <div className="p-12">
              <div className="space-y-6">
                <div className="text-[11px] font-light text-stone-900 dark:text-stone-300">{error}</div>
                <a
                  href="/login"
                  className="block w-full h-12 bg-gradient-to-r from-stone-800 to-stone-950 text-stone-50 text-[11px] tracking-[0.15em] uppercase font-light hover:from-amber-900 hover:to-amber-950 transition-all duration-300 shadow-[0_2px_8px_rgba(120,53,15,0.12)] dark:from-stone-700 dark:to-stone-800 dark:hover:from-amber-600 dark:hover:to-orange-600 dark:shadow-[0_2px_8px_rgba(0,0,0,0.3)] text-center leading-12"
                >
                  Back to Login
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 flex items-center justify-center p-8">
      {/* Judd-inspired geometric container */}
      <div className="w-full max-w-[480px]">
        {/* Precise rectangular form */}
        <div className="bg-white/95 backdrop-blur-sm border border-stone-800/90 shadow-[0_8px_32px_rgba(120,53,15,0.08)] dark:bg-stone-900/95 dark:border-stone-700/90 dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]">
          {/* Header - stark typography */}
          <div className="border-b border-stone-800/80 dark:border-stone-600/80 p-12">
            <h1 className="text-[13px] font-light tracking-[0.2em] uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
              Language Operator
              <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
            </h1>
          </div>

          {/* Content area - generous spacing */}
          <div className="p-12">
            {/* Form - vertical rhythm */}
            <form onSubmit={handleSignup} className="space-y-6">
              <div className="space-y-2">
                <label
                  htmlFor="name"
                  className="block text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400"
                >
                  Name
                </label>
                <input
                  id="name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={isLoading}
                  required
                  className="w-full h-12 border border-stone-200 px-4 text-sm font-light focus:outline-none focus:border-amber-900/40 focus:ring-1 focus:ring-amber-900/20 transition-all disabled:opacity-50 bg-stone-50/30 dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:focus:border-amber-600/60 dark:focus:ring-amber-600/30"
                />
              </div>

              <div className="space-y-2">
                <label
                  htmlFor="email"
                  className="block text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400"
                >
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={isLoading}
                  required
                  className="w-full h-12 border border-stone-200 px-4 text-sm font-light focus:outline-none focus:border-amber-900/40 focus:ring-1 focus:ring-amber-900/20 transition-all disabled:opacity-50 bg-stone-50/30 dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:focus:border-amber-600/60 dark:focus:ring-amber-600/30"
                />
              </div>

              <div className="space-y-2">
                <label
                  htmlFor="password"
                  className="block text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400"
                >
                  Password
                </label>
                <input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  disabled={isLoading}
                  required
                  minLength={8}
                  className="w-full h-12 border border-stone-200 px-4 text-sm font-light focus:outline-none focus:border-amber-900/40 focus:ring-1 focus:ring-amber-900/20 transition-all disabled:opacity-50 bg-stone-50/30 dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:focus:border-amber-600/60 dark:focus:ring-amber-600/30"
                />
              </div>

              {error && (
                <div className="text-[11px] font-light text-stone-900 dark:text-stone-300">{error}</div>
              )}

              <button
                type="submit"
                disabled={isLoading}
                className="w-full h-12 bg-gradient-to-r from-stone-800 to-stone-950 text-stone-50 text-[11px] tracking-[0.15em] uppercase font-light hover:from-amber-900 hover:to-amber-950 transition-all duration-300 disabled:opacity-50 shadow-[0_2px_8px_rgba(120,53,15,0.12)] dark:from-stone-700 dark:to-stone-800 dark:hover:from-amber-600 dark:hover:to-orange-600 dark:shadow-[0_2px_8px_rgba(0,0,0,0.3)]"
              >
                {isLoading ? 'Processing' : 'Create Account'}
              </button>
            </form>
          </div>

          {/* Footer */}
          <div className="border-t border-stone-800/80 dark:border-stone-600/80 p-12">
            <a
              href="/login"
              className="block text-[11px] tracking-[0.15em] uppercase font-light text-stone-600 hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500 transition-colors"
            >
              Sign In
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function SignupPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 flex items-center justify-center p-8">
        <div className="w-full max-w-[480px]">
          <div className="bg-white/95 backdrop-blur-sm border border-stone-800/90 shadow-[0_8px_32px_rgba(120,53,15,0.08)] dark:bg-stone-900/95 dark:border-stone-700/90 dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]">
            <div className="border-b border-stone-800/80 dark:border-stone-600/80 p-12">
              <h1 className="text-[13px] font-light tracking-[0.2em] uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
                Language Operator
                <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
              </h1>
            </div>
            <div className="p-12 flex justify-center">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-stone-900 dark:border-amber-400"></div>
            </div>
          </div>
        </div>
      </div>
    }>
      <SignupForm />
    </Suspense>
  )
}
