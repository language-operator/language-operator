import { Metadata } from 'next'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { SettingsNav } from './settings-nav'

export const metadata: Metadata = {
  title: 'Settings - Language Operator Dashboard',
  description: 'Manage your account and organization settings',
}

interface SettingsLayoutProps {
  children: React.ReactNode
}

export default function SettingsLayout({ children }: SettingsLayoutProps) {
  return (
    <AuthenticatedLayout>
      <div className="flex flex-1 gap-6">
        <SettingsNav />
        <div className="flex-1">
          {children}
        </div>
      </div>
    </AuthenticatedLayout>
  )
}