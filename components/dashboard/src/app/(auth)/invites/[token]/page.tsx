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
      router.push(`/settings/organizations/${invite.organization.id}`)
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
        <Card className="w-full max-w-md mx-auto">
          <CardHeader className="text-center">
            <div className="mx-auto w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mb-4">
              <Building2 className="w-6 h-6 text-blue-600" />
            </div>
            <CardTitle>You're Invited!</CardTitle>
            <CardDescription>
              You&apos;ve been invited to join <strong>{invite.organization.name}</strong>
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="border rounded-lg p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Organization:</span>
                <span className="text-sm text-gray-600">{invite.organization.name}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Your Role:</span>
                <Badge className={`${getRoleColor(invite.role)} flex items-center gap-1`}>
                  {getRoleIcon(invite.role)}
                  {invite.role}
                </Badge>
              </div>
              <div className="text-xs text-gray-500 pt-2 border-t">
                {getRoleDescription(invite.role)}
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Invited Email:</span>
                <span className="text-sm text-gray-600">{invite.email}</span>
              </div>
            </div>

            {!session?.user ? (
              <div className="space-y-3">
                <p className="text-sm text-center text-gray-600">
                  Sign in with your account to accept this invitation
                </p>
                <Button
                  onClick={() => signIn(undefined, { 
                    callbackUrl: `/invites/${token}` 
                  })}
                  className="w-full"
                  disabled={accepting}
                >
                  Sign In to Accept
                </Button>
              </div>
            ) : (
              <div className="flex gap-3">
                <Button
                  variant="outline"
                  onClick={handleDeclineInvite}
                  className="flex-1"
                  disabled={accepting}
                >
                  Decline
                </Button>
                <Button
                  onClick={handleAcceptInvite}
                  className="flex-1"
                  disabled={accepting}
                >
                  {accepting ? 'Joining...' : 'Accept Invitation'}
                </Button>
              </div>
            )}

            <p className="text-xs text-center text-gray-500">
              Expires on {new Date(invite.expiresAt).toLocaleDateString()}
            </p>
          </CardContent>
        </Card>
      )
    }

    return null
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {renderContent()}
      </div>
    </div>
  )
}