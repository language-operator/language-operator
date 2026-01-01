'use client'

import { useEffect, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { useSession, signIn } from 'next-auth/react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Building2, Crown, Shield, Edit, Eye, AlertCircle, CheckCircle, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import Link from 'next/link'

interface OrganizationInvite {
  id: string
  email: string
  role: string
  expiresAt: string
  createdAt: string
  organization: {
    id: string
    name: string
    slug: string
  }
}

type InviteStatus = 'loading' | 'valid' | 'expired' | 'invalid' | 'wrong-email' | 'already-member'

export default function InviteAcceptancePage() {
  const params = useParams()
  const router = useRouter()
  const { data: session, status: sessionStatus } = useSession()
  const token = params.token as string

  const [invite, setInvite] = useState<OrganizationInvite | null>(null)
  const [inviteStatus, setInviteStatus] = useState<InviteStatus>('loading')
  const [accepting, setAccepting] = useState(false)

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'owner': return <Crown className="w-5 h-5" />
      case 'admin': return <Shield className="w-5 h-5" />
      case 'editor': return <Edit className="w-5 h-5" />
      case 'viewer': return <Eye className="w-5 h-5" />
      default: return <Eye className="w-5 h-5" />
    }
  }

  const getRoleColor = (role: string) => {
    switch (role) {
      case 'owner': return 'bg-purple-100 text-purple-800'
      case 'admin': return 'bg-blue-100 text-blue-800'
      case 'editor': return 'bg-green-100 text-green-800'
      case 'viewer': return 'bg-gray-100 text-gray-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  const getRoleDescription = (role: string) => {
    switch (role) {
      case 'owner': return 'Full access including billing and organization management'
      case 'admin': return 'Manage members and all resources'
      case 'editor': return 'Create and edit resources'
      case 'viewer': return 'Read-only access to resources'
      default: return ''
    }
  }

  // Validate invitation on load
  useEffect(() => {
    const validateInvite = async () => {
      if (!token) {
        setInviteStatus('invalid')
        return
      }

      try {
        const response = await fetch(`/api/invites/${token}`)
        
        if (!response.ok) {
          if (response.status === 404) {
            setInviteStatus('invalid')
          } else if (response.status === 410) {
            setInviteStatus('expired')
          } else {
            setInviteStatus('invalid')
          }
          return
        }

        const data = await response.json()
        setInvite(data.invite)
        
        // If user is logged in, check email match
        if (session?.user?.email) {
          if (data.invite.email !== session.user.email) {
            setInviteStatus('wrong-email')
          } else {
            setInviteStatus('valid')
          }
        } else {
          setInviteStatus('valid')
        }
      } catch (error) {
        console.error('Error validating invite:', error)
        setInviteStatus('invalid')
      }
    }

    if (sessionStatus !== 'loading') {
      validateInvite()
    }
  }, [token, session, sessionStatus])

  const handleAcceptInvite = async () => {
    if (!session?.user?.email) {
      // Redirect to sign in with invite token
      await signIn(undefined, { 
        callbackUrl: `/invites/${token}` 
      })
      return
    }

    if (inviteStatus !== 'valid' || !invite) return

    setAccepting(true)

    try {
      const response = await fetch('/api/invites/accept', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ token }),
      })

      const data = await response.json()

      if (!response.ok) {
        if (response.status === 409 && data.message?.includes('already a member')) {
          setInviteStatus('already-member')
        } else {
          throw new Error(data.error || 'Failed to accept invitation')
        }
        return
      }

      toast.success(`Successfully joined ${invite.organization.name}!`)
      router.push('/')
    } catch (error: unknown) {
      console.error('Error accepting invite:', error)
      toast.error((error as Error).message || 'Failed to accept invitation')
    } finally {
      setAccepting(false)
    }
  }

  const handleDeclineInvite = () => {
    router.push('/')
  }

  const renderContent = () => {
    if (inviteStatus === 'loading') {
      return (
        <Card className="w-full max-w-md mx-auto">
          <CardContent className="flex items-center justify-center p-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </CardContent>
        </Card>
      )
    }

    if (inviteStatus === 'invalid') {
      return (
        <Card className="w-full max-w-md mx-auto">
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mb-4">
              <XCircle className="w-6 h-6 text-red-600" />
            </div>
            <CardTitle className="text-red-900">Invalid Invitation</CardTitle>
            <CardDescription>
              This invitation link is not valid or has been cancelled.
            </CardDescription>
          </CardHeader>
          <CardContent className="text-center">
            <Link href="/">
              <Button variant="outline" className="w-full">
                Return to Dashboard
              </Button>
            </Link>
          </CardContent>
        </Card>
      )
    }

    if (inviteStatus === 'expired') {
      return (
        <Card className="w-full max-w-md mx-auto">
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-yellow-100 rounded-full flex items-center justify-center mb-4">
              <AlertCircle className="w-6 h-6 text-yellow-600" />
            </div>
            <CardTitle className="text-yellow-900">Invitation Expired</CardTitle>
            <CardDescription>
              This invitation has expired. Please contact the organization admin for a new invitation.
            </CardDescription>
          </CardHeader>
          <CardContent className="text-center">
            <Link href="/">
              <Button variant="outline" className="w-full">
                Return to Dashboard
              </Button>
            </Link>
          </CardContent>
        </Card>
      )
    }

    if (inviteStatus === 'wrong-email') {
      return (
        <Card className="w-full max-w-md mx-auto">
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-yellow-100 rounded-full flex items-center justify-center mb-4">
              <AlertCircle className="w-6 h-6 text-yellow-600" />
            </div>
            <CardTitle className="text-yellow-900">Email Mismatch</CardTitle>
            <CardDescription>
              This invitation is for {invite?.email}, but you&apos;re signed in as {session?.user?.email}.
              Please sign in with the correct account or contact the organization admin.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <Button
              variant="outline"
              className="w-full"
              onClick={() => signIn()}
            >
              Sign In with Different Account
            </Button>
            <Link href="/">
              <Button variant="ghost" className="w-full">
                Return to Dashboard
              </Button>
            </Link>
          </CardContent>
        </Card>
      )
    }

    if (inviteStatus === 'already-member') {
      return (
        <Card className="w-full max-w-md mx-auto">
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mb-4">
              <CheckCircle className="w-6 h-6 text-green-600" />
            </div>
            <CardTitle className="text-green-900">Already a Member</CardTitle>
            <CardDescription>
              You&apos;re already a member of {invite?.organization.name}.
            </CardDescription>
          </CardHeader>
          <CardContent className="text-center">
            <Link href={`/settings/organizations/${invite?.organization.id}`}>
              <Button className="w-full">
                Go to Organization
              </Button>
            </Link>
          </CardContent>
        </Card>
      )
    }

    if (inviteStatus === 'valid' && invite) {
      return (
        /* Precise rectangular form */
        <div className="bg-white/95 backdrop-blur-sm border border-stone-800/90 shadow-[0_8px_32px_rgba(120,53,15,0.08)] dark:bg-stone-900/95 dark:border-stone-700/90 dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]">
          {/* Header - stark typography */}
          <div className="border-b border-stone-800/80 dark:border-stone-600/80 p-12">
            <h1 className="text-[13px] font-light tracking-[0.2em] uppercase text-stone-900 dark:text-stone-300 flex items-center gap-1">
              Language Operator
              <span className="inline-block w-2 h-3.5 bg-stone-900 dark:bg-amber-400 animate-pulse" />
            </h1>
          </div>

          {/* Content area - generous spacing */}
          <div className="p-12 space-y-8">
            {/* Invitation header */}
            <div>
              <h2 className="text-[11px] font-light tracking-[0.15em] uppercase text-stone-900 dark:text-stone-300">
                Accept Invitation
              </h2>
            </div>

            {/* Properties - clean vertical rhythm */}
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400">
                  Organization
                </span>
                <span className="text-sm font-light text-stone-900 dark:text-stone-300">
                  {invite.organization.name}
                </span>
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400">
                  Your Role
                </span>
                <span className="text-sm font-light text-stone-900 dark:text-stone-300 flex items-center gap-2">
                  {getRoleIcon(invite.role)}
                  {invite.role}
                </span>
              </div>
              
              <div className="flex justify-between items-center">
                <span className="text-[10px] tracking-[0.2em] uppercase font-light text-stone-600 dark:text-stone-400">
                  Email
                </span>
                <span className="text-sm font-light text-stone-600 dark:text-stone-400">
                  {invite.email}
                </span>
              </div>
            </div>

            {/* Actions */}
            {!session?.user ? (
              <div className="space-y-3">
                  <button
                    onClick={() => router.push(`/signup?callbackUrl=${encodeURIComponent(`/invites/${token}`)}&email=${encodeURIComponent(invite.email)}`)}
                    disabled={accepting}
                    className="w-full h-12 bg-gradient-to-r from-stone-800 to-stone-950 text-stone-50 text-[11px] tracking-[0.15em] uppercase font-light hover:from-amber-900 hover:to-amber-950 transition-all duration-300 disabled:opacity-50 shadow-[0_2px_8px_rgba(120,53,15,0.12)] dark:from-stone-700 dark:to-stone-800 dark:hover:from-amber-600 dark:hover:to-orange-600 dark:shadow-[0_2px_8px_rgba(0,0,0,0.3)]"
                  >
                    Create Account
                  </button>
                  <button
                    onClick={() => signIn(undefined, { callbackUrl: `/invites/${token}` })}
                    disabled={accepting}
                    className="w-full h-12 border border-stone-200 text-stone-600 text-[11px] tracking-[0.15em] uppercase font-light hover:border-amber-900/40 hover:text-amber-900 transition-all disabled:opacity-50 bg-stone-50/30 dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:hover:border-amber-600/60 dark:hover:text-amber-600"
                  >
                    Sign In with Existing Account
                  </button>
              </div>
            ) : (
              <div className="space-y-3">
                <button
                  onClick={handleAcceptInvite}
                  disabled={accepting}
                  className="w-full h-12 bg-gradient-to-r from-stone-800 to-stone-950 text-stone-50 text-[11px] tracking-[0.15em] uppercase font-light hover:from-amber-900 hover:to-amber-950 transition-all duration-300 disabled:opacity-50 shadow-[0_2px_8px_rgba(120,53,15,0.12)] dark:from-stone-700 dark:to-stone-800 dark:hover:from-amber-600 dark:hover:to-orange-600 dark:shadow-[0_2px_8px_rgba(0,0,0,0.3)]"
                >
                  {accepting ? 'Joining...' : 'Accept Invitation'}
                </button>
                <button
                  onClick={handleDeclineInvite}
                  disabled={accepting}
                  className="w-full h-12 border border-stone-200 text-stone-600 text-[11px] tracking-[0.15em] uppercase font-light hover:border-amber-900/40 hover:text-amber-900 transition-all disabled:opacity-50 bg-stone-50/30 dark:bg-stone-800/30 dark:border-stone-600 dark:text-stone-300 dark:hover:border-amber-600/60 dark:hover:text-amber-600"
                >
                  Decline
                </button>
              </div>
            )}

            {/* Footer note */}
            <p className="text-[10px] font-light text-center text-stone-500 dark:text-stone-500 tracking-[0.1em]">
              Expires {new Date(invite.expiresAt).toLocaleDateString()}
            </p>
          </div>
        </div>
      )
    }

    return null
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100 dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950 flex items-center justify-center p-8">
      {/* Judd-inspired geometric container */}
      <div className="w-full max-w-[480px]">
        {renderContent()}
      </div>
    </div>
  )
}