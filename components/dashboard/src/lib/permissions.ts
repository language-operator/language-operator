import { db } from './db'

export type Permission = 'view' | 'create' | 'edit' | 'delete' | 'manage_members' | 'delete_org' | 'manage_billing'
export type Role = 'owner' | 'admin' | 'editor' | 'viewer'

const rolePermissions: Record<Role, Permission[]> = {
  owner: ['view', 'create', 'edit', 'delete', 'manage_members', 'delete_org', 'manage_billing'],
  admin: ['view', 'create', 'edit', 'delete', 'manage_members'],
  editor: ['view', 'create', 'edit', 'delete'],
  viewer: ['view'],
}

/**
 * Check if a user has a specific permission in an organization
 */
export async function requirePermission(
  userId: string,
  organizationId: string,
  permission: Permission
): Promise<boolean> {
  const member = await db.organizationMember.findUnique({
    where: { organizationId_userId: { organizationId, userId } },
  })

  if (!member) return false

  const permissions = rolePermissions[member.role as Role]
  return permissions?.includes(permission) ?? false
}

/**
 * Get all organizations a user is a member of
 */
export async function getUserOrganizations(userId: string) {
  return db.organizationMember.findMany({
    where: { userId },
    include: {
      organization: true,
    },
  })
}

/**
 * Check if a user can access a specific Kubernetes namespace
 */
export async function canUserAccessNamespace(userId: string, namespace: string): Promise<boolean> {
  const org = await db.organization.findUnique({
    where: { namespace },
    include: {
      members: {
        where: { userId },
      },
    },
  })

  return org !== null && org.members.length > 0
}

/**
 * Get user's role in an organization
 */
export async function getUserRole(userId: string, organizationId: string): Promise<Role | null> {
  const member = await db.organizationMember.findUnique({
    where: { organizationId_userId: { organizationId, userId } },
  })

  return member?.role as Role | null
}
