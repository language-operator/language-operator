import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'standalone',
  // Disable type checking during build (will be done in CI)
  typescript: {
    ignoreBuildErrors: false,
  },
  eslint: {
    ignoreDuringBuilds: false,
  },
};

export default nextConfig;
