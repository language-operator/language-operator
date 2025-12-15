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
  transpilePackages: ['react-syntax-highlighter']
};

export default nextConfig;
