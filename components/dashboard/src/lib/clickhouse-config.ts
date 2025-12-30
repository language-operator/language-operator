import { createClient } from '@clickhouse/client'

/**
 * Get ClickHouse client configuration based on environment
 */
export function getClickHouseConfig() {
  // Check if we should use direct ClickHouse access (bypasses kubectl proxy auth issues)
  if (process.env.CLICKHOUSE_DIRECT_ACCESS === 'true') {
    const host = process.env.CLICKHOUSE_HOST || 'localhost:8123'
    return {
      url: `http://${host}`,
      database: process.env.CLICKHOUSE_DATABASE,
      username: process.env.CLICKHOUSE_USERNAME,
      password: process.env.CLICKHOUSE_PASSWORD,
    }
  }
  
  // Check if we're in docker-compose mode with kubectl-proxy
  if (process.env.KUBERNETES_SERVER_URL?.includes('kubectl-proxy')) {
    // Use kubectl-proxy to connect to ClickHouse in language-operator namespace
    // Use url format to avoid deprecation warnings about host
    return {
      url: `${process.env.KUBERNETES_SERVER_URL}/api/v1/namespaces/language-operator/services/language-operator-clickhouse:8123/proxy`,
      database: process.env.CLICKHOUSE_DATABASE,
      username: process.env.CLICKHOUSE_USERNAME,
      password: process.env.CLICKHOUSE_PASSWORD,
    }
  }
  
  // Default to environment variable or localhost
  return {
    url: process.env.CLICKHOUSE_URL || 'http://localhost:8123',
    database: process.env.CLICKHOUSE_DATABASE,
    username: process.env.CLICKHOUSE_USERNAME,
    password: process.env.CLICKHOUSE_PASSWORD,
  }
}

/**
 * Create a ClickHouse client with environment-appropriate configuration
 */
export function createClickHouseClient() {
  return createClient(getClickHouseConfig())
}