import { createClient } from '@clickhouse/client'

/**
 * Get ClickHouse client configuration based on environment
 */
export function getClickHouseConfig() {
  // Check if we're in docker-compose mode with kubectl-proxy
  if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy')) {
    // Use kubectl-proxy to connect to ClickHouse in language-operator namespace
    // Use url format to avoid deprecation warnings about host
    return {
      url: `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`,
      database: 'langop',
    }
  }
  
  // Default to environment variable or localhost
  return {
    host: process.env.CLICKHOUSE_URL || 'http://localhost:8123',
    database: 'langop',
  }
}

/**
 * Create a ClickHouse client with environment-appropriate configuration
 */
export function createClickHouseClient() {
  return createClient(getClickHouseConfig())
}