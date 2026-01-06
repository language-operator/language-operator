import { redirect } from 'next/navigation'

interface SettingsPageProps {
  params: Promise<{ org_id: string }>
}

export default async function SettingsPage({ params }: SettingsPageProps) {
  const resolvedParams = await params
  const orgId = resolvedParams.org_id
  redirect(`/${orgId}/settings/organizations`)
}