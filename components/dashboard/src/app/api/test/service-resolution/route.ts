import { NextRequest, NextResponse } from 'next/server'
import { serviceResolver } from '@/lib/service-resolver'

// GET /api/test/service-resolution - Test endpoint to show service URL resolution
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const serviceName = searchParams.get('service') || 'qwen3-coder-30b'
  const namespace = searchParams.get('namespace') || 'org-2h6mnkxh'
  const port = parseInt(searchParams.get('port') || '8000')

  const envInfo = serviceResolver.getEnvironmentInfo()
  
  // Test different service URL resolutions
  const results = {
    environment: envInfo,
    serviceUrl: serviceResolver.resolveServiceUrl({ 
      serviceName, 
      namespace, 
      port 
    }),
    agentChatUrl: serviceResolver.resolveAgentChatUrl(serviceName, namespace, 80),
    modelUrl: serviceResolver.resolveModelUrl(serviceName, namespace, 8000),
    
    // Show what the URLs would be in different environments
    comparison: {
      kubernetes: `http://${serviceName}.${namespace}.svc.cluster.local:${port}`,
      dockerCompose: `http://kubectl-proxy:8001/api/v1/namespaces/${namespace}/services/${serviceName}:${port}/proxy`,
    },
    
    // Environment variables that affect resolution
    environmentVariables: {
      KUBERNETES_SERVICE_HOST: process.env.KUBERNETES_SERVICE_HOST,
      KUBERNETES_SERVER_URL: process.env.KUBERNETES_SERVER_URL,
      KUBECONFIG: process.env.KUBECONFIG,
      NODE_ENV: process.env.NODE_ENV
    }
  }

  return NextResponse.json(results, { 
    headers: {
      'Content-Type': 'application/json'
    }
  })
}