import 'next-auth'

declare module 'next-auth' {
  interface Session {
    user: {
      id: string
      email: string
      name?: string | null
      image?: string | null
    }
    organizations: Array<{
      id: string
      name: string
      slug: string
      namespace: string
      role: string
    }>
    activeOrganization: {
      id: string
      name: string
      slug: string
      namespace: string
      role: string
    } | null
  }
}
