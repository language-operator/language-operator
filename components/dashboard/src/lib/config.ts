export const config = {
  isSaaS: process.env.NEXT_PUBLIC_SAAS_MODE === 'true',
  signozUrl: process.env.SIGNOZ_URL,
  prometheusUrl: process.env.PROMETHEUS_URL,
  dashboardUrl: process.env.NEXT_PUBLIC_DASHBOARD_URL || process.env.NEXTAUTH_URL || 'http://localhost:3000',

  // Auth providers
  auth: {
    googleClientId: process.env.GOOGLE_CLIENT_ID,
    googleClientSecret: process.env.GOOGLE_CLIENT_SECRET,
    githubClientId: process.env.GITHUB_CLIENT_ID,
    githubClientSecret: process.env.GITHUB_CLIENT_SECRET,
  },

  // Feature flags
  features: {
    emailPassword: process.env.NEXT_PUBLIC_EMAIL_AUTH_ENABLED === 'true',
    billing: process.env.NEXT_PUBLIC_BILLING_ENABLED === 'true',
    invites: process.env.NEXT_PUBLIC_INVITES_ENABLED === 'true',
  },

  // Limits
  limits: {
    freeOrgLimit: parseInt(process.env.FREE_ORG_LIMIT || '1'), // Orgs per user
  },
} as const
