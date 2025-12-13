#!/usr/bin/env node

/**
 * CRUD Completeness Verification Script
 * 
 * Systematically verifies that all 8 required routing files exist for a given resource
 * to prevent false completion reports on CRUD implementations.
 * 
 * Usage:
 *   node scripts/verify-crud-completeness.js --resource=tools
 *   node scripts/verify-crud-completeness.js --resource=all
 *   node scripts/verify-crud-completeness.js --help
 */

const fs = require('fs');
const path = require('path');

// Configuration
const DASHBOARD_ROOT = path.join(__dirname, '../components/dashboard');
const APP_DIR = path.join(DASHBOARD_ROOT, 'src/app');

// Define the 8 required routing patterns for complete CRUD implementation
const ROUTING_PATTERNS = [
  // Global routes (secondary user path)
  { type: 'global', path: '{resource}/new/page.tsx', description: 'Global create page' },
  { type: 'global', path: '{resource}/[{id}]/page.tsx', description: 'Global detail page' },
  { type: 'global', path: '{resource}/[{id}]/edit/page.tsx', description: 'Global edit page' },
  
  // Cluster-scoped routes (primary user path - most often missing)
  { type: 'cluster', path: 'clusters/[name]/{resource}/page.tsx', description: 'Cluster resource list' },
  { type: 'cluster', path: 'clusters/[name]/{resource}/new/page.tsx', description: 'Cluster create page ⚠️' },
  { type: 'cluster', path: 'clusters/[name]/{resource}/[{id}]/page.tsx', description: 'Cluster detail page ⚠️' },
  { type: 'cluster', path: 'clusters/[name]/{resource}/[{id}]/edit/page.tsx', description: 'Cluster edit page ⚠️' },
  
  // API routes (backend)
  { type: 'api', path: 'api/{resource}/route.ts', description: 'Resource API endpoints' }
];

// Resource configurations
const RESOURCES = {
  agents: { plural: 'agents', singular: 'agent', id: 'agentName' },
  models: { plural: 'models', singular: 'model', id: 'modelName' },
  tools: { plural: 'tools', singular: 'tool', id: 'toolName' },
  personas: { plural: 'personas', singular: 'persona', id: 'personaName' },
  clusters: { plural: 'clusters', singular: 'cluster', id: 'clusterName' }
};

function parseArgs() {
  const args = process.argv.slice(2);
  const options = {};
  
  for (const arg of args) {
    if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    } else if (arg.startsWith('--resource=')) {
      options.resource = arg.split('=')[1];
    } else if (arg.startsWith('--format=')) {
      options.format = arg.split('=')[1];
    }
  }
  
  return options;
}

function printHelp() {
  console.log(`
CRUD Completeness Verification Script

USAGE:
  node scripts/verify-crud-completeness.js [OPTIONS]

OPTIONS:
  --resource=<name>    Check specific resource (agents, models, tools, personas, clusters)
  --resource=all       Check all resources
  --format=json        Output results in JSON format
  --help, -h          Show this help message

EXAMPLES:
  node scripts/verify-crud-completeness.js --resource=tools
  node scripts/verify-crud-completeness.js --resource=all
  node scripts/verify-crud-completeness.js --resource=agents --format=json

DESCRIPTION:
  This script verifies that all 8 required routing files exist for complete CRUD
  implementation. It prevents false completion reports where agents claim 100%
  completion but users experience 404 errors on cluster-scoped workflows.

  The 8 required files per resource:
  1. Global create page
  2. Global detail page  
  3. Global edit page
  4. Cluster list page
  5. Cluster create page (most often missing)
  6. Cluster detail page (most often missing)
  7. Cluster edit page (most often missing)
  8. API routes

CONTEXT:
  This addresses Issue #122 where CRUD implementations are falsely reported as
  complete when critical cluster-scoped routing files are missing, causing
  user-facing 404 errors despite functional backend APIs and components.
`);
}

function verifyResource(resourceName) {
  const resource = RESOURCES[resourceName];
  if (!resource) {
    throw new Error(`Unknown resource: ${resourceName}. Available: ${Object.keys(RESOURCES).join(', ')}`);
  }

  const results = {
    resource: resourceName,
    total: ROUTING_PATTERNS.length,
    found: 0,
    missing: [],
    existing: [],
    completionPercent: 0
  };

  for (const pattern of ROUTING_PATTERNS) {
    const filePath = pattern.path
      .replace('{resource}', resource.plural)
      .replace('{id}', resource.id);
    
    const fullPath = path.join(APP_DIR, filePath);
    const exists = fs.existsSync(fullPath);
    
    const fileResult = {
      type: pattern.type,
      path: filePath,
      fullPath,
      description: pattern.description,
      exists,
      critical: pattern.description.includes('⚠️')
    };
    
    if (exists) {
      results.found++;
      results.existing.push(fileResult);
    } else {
      results.missing.push(fileResult);
    }
  }
  
  results.completionPercent = Math.round((results.found / results.total) * 100);
  results.isComplete = results.missing.length === 0;
  results.hasClusterRoutes = results.existing.some(f => f.type === 'cluster' && f.path.includes('/new/'));
  
  return results;
}

function formatResults(results, format = 'human') {
  if (format === 'json') {
    return JSON.stringify(results, null, 2);
  }
  
  let output = '';
  
  // Header
  const status = results.isComplete ? '✅ COMPLETE' : '⚠️  INCOMPLETE';
  const completion = `${results.found}/${results.total}`;
  
  output += `\n${status} ${results.resource.toUpperCase()} - ${completion} files (${results.completionPercent}%)\n`;
  output += '═'.repeat(60) + '\n';
  
  // Existing files
  if (results.existing.length > 0) {
    output += '\n✅ EXISTING FILES:\n';
    for (const file of results.existing) {
      const typeLabel = file.type.toUpperCase().padEnd(8);
      output += `  ${typeLabel} ${file.path}\n`;
    }
  }
  
  // Missing files
  if (results.missing.length > 0) {
    output += '\n❌ MISSING FILES:\n';
    for (const file of results.missing) {
      const typeLabel = file.type.toUpperCase().padEnd(8);
      const critical = file.critical ? ' [CRITICAL]' : '';
      output += `  ${typeLabel} ${file.path}${critical}\n`;
      output += `           ${file.description}\n`;
    }
  }
  
  // Analysis
  output += '\n📊 ANALYSIS:\n';
  
  if (results.isComplete) {
    output += '  ✅ All routing patterns implemented\n';
    output += '  ✅ Users can create resources from cluster pages\n';
    output += '  ✅ No 404 errors expected\n';
  } else {
    const missingCluster = results.missing.filter(f => f.type === 'cluster');
    const missingGlobal = results.missing.filter(f => f.type === 'global');
    const missingApi = results.missing.filter(f => f.type === 'api');
    
    if (missingCluster.length > 0) {
      output += `  ⚠️  ${missingCluster.length} cluster-scoped routes missing (PRIMARY USER PATH)\n`;
      output += '     Users will experience 404s when creating resources from cluster pages\n';
    }
    
    if (missingGlobal.length > 0) {
      output += `  ⚠️  ${missingGlobal.length} global routes missing\n`;
    }
    
    if (missingApi.length > 0) {
      output += `  ⚠️  ${missingApi.length} API routes missing\n`;
    }
    
    output += `\n  This represents a ${100 - results.completionPercent}% gap in CRUD implementation.\n`;
  }
  
  output += '\n';
  return output;
}

function main() {
  try {
    const options = parseArgs();
    
    if (!options.resource) {
      console.error('Error: --resource parameter required');
      console.error('Run --help for usage information');
      process.exit(1);
    }
    
    if (!fs.existsSync(APP_DIR)) {
      console.error(`Error: Dashboard app directory not found: ${APP_DIR}`);
      console.error('Make sure you\'re running this from the language-operator root directory');
      process.exit(1);
    }
    
    if (options.resource === 'all') {
      const allResults = {};
      let totalComplete = 0;
      
      console.log('🔍 CRUD COMPLETENESS VERIFICATION - ALL RESOURCES');
      console.log('═'.repeat(60));
      
      for (const resourceName of Object.keys(RESOURCES)) {
        const results = verifyResource(resourceName);
        allResults[resourceName] = results;
        
        if (results.isComplete) totalComplete++;
        
        console.log(formatResults(results, options.format));
      }
      
      // Summary
      const totalResources = Object.keys(RESOURCES).length;
      console.log('\n📋 SUMMARY:');
      console.log(`Complete: ${totalComplete}/${totalResources} resources`);
      console.log(`Overall: ${Math.round((totalComplete / totalResources) * 100)}%`);
      
      if (options.format === 'json') {
        console.log(JSON.stringify(allResults, null, 2));
      }
      
    } else {
      const results = verifyResource(options.resource);
      console.log('🔍 CRUD COMPLETENESS VERIFICATION');
      console.log(formatResults(results, options.format));
    }
    
    // Exit with error code if any resource is incomplete
    const hasIncomplete = options.resource === 'all' 
      ? Object.keys(RESOURCES).some(name => !verifyResource(name).isComplete)
      : !verifyResource(options.resource).isComplete;
    
    process.exit(hasIncomplete ? 1 : 0);
    
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = {
  verifyResource,
  RESOURCES,
  ROUTING_PATTERNS
};