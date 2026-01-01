'use client'

import { signIn } from 'next-auth/react'
import { useRouter } from 'next/navigation'
import { useState, useEffect } from 'react'

export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [signupEnabled, setSignupEnabled] = useState(true)

  // Check if signup is enabled
  useEffect(() => {
    const checkSignupEnabled = async () => {
      try {
        const response = await fetch('/api/auth/signup-enabled')
        const data = await response.json()
        setSignupEnabled(data.signupEnabled)
      } catch (error) {
        console.error('Error checking signup status:', error)
        // Default to enabled on error
        setSignupEnabled(true)
      }
    }
    
    checkSignupEnabled()
  }, [])

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError('')

    try {
      const result = await signIn('credentials', {
        email,
        password,
        redirect: false,
      })

      if (result?.error) {
        setError('Invalid email or password')
      } else {
        router.push('/')
        router.refresh()
      }
    } catch (err) {
      setError('An error occurred. Please try again.')
    } finally {
      setIsLoading(false)
    }
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
            <form onSubmit={handleEmailLogin} className="space-y-6">
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
                {isLoading ? 'Processing' : 'Sign In'}
              </button>
            </form>
          </div>

          {/* Footer */}
          {signupEnabled && (
            <div className="border-t border-stone-800/80 dark:border-stone-600/80 p-12">
              <a
                href="/signup"
                className="block text-[11px] tracking-[0.15em] uppercase font-light text-stone-600 hover:text-amber-900 dark:text-stone-400 dark:hover:text-amber-500 transition-colors"
              >
                Create Account
              </a>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
