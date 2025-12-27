import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'standalone',
  typescript: {
    ignoreBuildErrors: false,
  },
  skipTrailingSlashRedirect: true,
  generateBuildId: async () => {
    return 'build-' + Date.now()
  },
  transpilePackages: [
    'react-syntax-highlighter',
    '@evilmartians/agent-prism-data',
    '@evilmartians/agent-prism-types',
    '@clickhouse/client'
  ]
};

export default nextConfig;
