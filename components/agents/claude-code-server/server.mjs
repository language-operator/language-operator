/**
 * claude-code-server — A2A protocol server wrapping Claude Code
 *
 * Implements the Agent-to-Agent (A2A) protocol so any A2A-compatible
 * orchestrator or agent can call Claude Code agents directly.
 *
 * Endpoints:
 *   GET  /healthz                  → liveness/readiness probe
 *   GET  /.well-known/agent-card   → A2A agent capabilities metadata
 *   POST /                         → A2A task submission (streaming SSE)
 *
 * A2A spec: https://google.github.io/A2A/
 */

import express from 'express'
import { readFileSync, existsSync, mkdirSync, writeFileSync } from 'fs'
import { join } from 'path'
import { homedir } from 'os'
import { v4 as uuidv4 } from 'uuid'
import { query } from '@anthropic-ai/claude-code'

const PORT = parseInt(process.env.PORT ?? '8080', 10)
const HOME = process.env.HOME ?? homedir()
const WORKSPACE = process.env.WORKSPACE ?? '/workspace'
const TASK_STORE_DIR = join(WORKSPACE, '.a2a')
const AGENT_NAME = process.env.AGENT_NAME ?? 'claude-code-agent'
const MAX_TURNS = process.env.CLAUDE_CODE_MAX_TURNS
  ? parseInt(process.env.CLAUDE_CODE_MAX_TURNS, 10)
  : undefined

mkdirSync(TASK_STORE_DIR, { recursive: true })

// -------------------------------------------------------------------
// Agent card — loaded from adapter-written config if present
// -------------------------------------------------------------------
function loadAgentCard() {
  const configPath = join(HOME, '.claude', 'claude-a2a.config.json')
  if (existsSync(configPath)) {
    try {
      const cfg = JSON.parse(readFileSync(configPath, 'utf8'))
      return {
        name: cfg.name ?? AGENT_NAME,
        description: cfg.description ?? 'Claude Code agent',
      }
    } catch {
      // fall through to defaults
    }
  }
  return { name: AGENT_NAME, description: 'Claude Code agent' }
}

function buildAgentCard() {
  const { name, description } = loadAgentCard()
  return {
    name,
    description,
    url: `http://localhost:${PORT}/`,
    version: '1.0.0',
    capabilities: {
      streaming: true,
      pushNotifications: false,
    },
    skills: [
      {
        id: 'code',
        name: 'Code',
        description: 'Execute coding tasks, review code, and perform software engineering work.',
        inputModes: ['text'],
        outputModes: ['text'],
      },
    ],
  }
}

// -------------------------------------------------------------------
// Filesystem task store — tasks survive pod restarts via workspace PVC
// -------------------------------------------------------------------
function taskPath(taskId) {
  return join(TASK_STORE_DIR, `${taskId}.json`)
}

function saveTask(task) {
  writeFileSync(taskPath(task.id), JSON.stringify(task))
}

function loadTask(taskId) {
  const p = taskPath(taskId)
  if (!existsSync(p)) return null
  try {
    return JSON.parse(readFileSync(p, 'utf8'))
  } catch {
    return null
  }
}

// -------------------------------------------------------------------
// A2A message helpers
// -------------------------------------------------------------------
function sseEvent(res, eventType, data) {
  res.write(`event: ${eventType}\n`)
  res.write(`data: ${JSON.stringify(data)}\n\n`)
}

function taskStatusEvent(res, taskId, state, message) {
  sseEvent(res, 'task-status-update', {
    id: taskId,
    status: { state, ...(message ? { message: { role: 'agent', parts: [{ type: 'text', text: message }] } } : {}) },
    final: state === 'completed' || state === 'failed' || state === 'canceled',
  })
}

function artifactEvent(res, taskId, name, text) {
  sseEvent(res, 'artifact-update', {
    taskId,
    artifact: {
      artifactId: uuidv4(),
      name,
      parts: [{ type: 'text', text }],
    },
    append: false,
    lastChunk: true,
  })
}

// -------------------------------------------------------------------
// Express app
// -------------------------------------------------------------------
const app = express()
app.use(express.json())

app.get('/healthz', (_req, res) => {
  res.json({ status: 'ok' })
})

app.get('/.well-known/agent-card', (_req, res) => {
  res.json(buildAgentCard())
})

app.post('/', async (req, res) => {
  const body = req.body

  // Validate JSON-RPC envelope
  if (!body || body.jsonrpc !== '2.0' || !body.method) {
    return res.status(400).json({ error: 'Invalid A2A request' })
  }

  if (body.method === 'tasks/send' || body.method === 'tasks/sendSubscribe') {
    const params = body.params ?? {}
    const taskId = params.id ?? uuidv4()
    const contextId = params.contextId ?? taskId // map to sessionId for continuity
    const message = params.message

    if (!message || !Array.isArray(message.parts)) {
      return res.status(400).json({ error: 'Missing message.parts' })
    }

    const userText = message.parts
      .filter(p => p.type === 'text')
      .map(p => p.text)
      .join('\n')

    if (!userText) {
      return res.status(400).json({ error: 'No text content in message' })
    }

    // Streaming SSE response
    res.setHeader('Content-Type', 'text/event-stream')
    res.setHeader('Cache-Control', 'no-cache')
    res.setHeader('Connection', 'keep-alive')
    res.flushHeaders()

    const task = {
      id: taskId,
      contextId,
      status: { state: 'working' },
      createdAt: new Date().toISOString(),
    }
    saveTask(task)

    taskStatusEvent(res, taskId, 'working', null)

    try {
      const queryOptions = {
        prompt: userText,
        abortController: new AbortController(),
        options: {
          cwd: WORKSPACE,
          ...(MAX_TURNS !== undefined ? { maxTurns: MAX_TURNS } : {}),
        },
      }

      const outputParts = []

      for await (const sdkMessage of query(queryOptions)) {
        if (sdkMessage.type === 'assistant') {
          for (const block of sdkMessage.message.content ?? []) {
            if (block.type === 'text' && block.text) {
              outputParts.push(block.text)
            }
            if (block.type === 'tool_use') {
              // Surface tool invocations as interim status messages
              taskStatusEvent(res, taskId, 'working', `Using tool: ${block.name}`)
            }
          }
        } else if (sdkMessage.type === 'result') {
          if (sdkMessage.subtype === 'success') {
            const finalText = outputParts.join('\n') || (sdkMessage.result ?? '')
            artifactEvent(res, taskId, 'response', finalText)
            task.status = { state: 'completed' }
            saveTask(task)
            taskStatusEvent(res, taskId, 'completed', null)
          } else {
            task.status = { state: 'failed' }
            saveTask(task)
            taskStatusEvent(res, taskId, 'failed', sdkMessage.error ?? 'Task failed')
          }
        }
      }
    } catch (err) {
      console.error(`Task ${taskId} error:`, err)
      task.status = { state: 'failed' }
      saveTask(task)
      taskStatusEvent(res, taskId, 'failed', err.message ?? 'Internal error')
    }

    res.end()
    return
  }

  if (body.method === 'tasks/get') {
    const taskId = body.params?.id
    if (!taskId) {
      return res.status(400).json({ error: 'Missing task id' })
    }
    const task = loadTask(taskId)
    if (!task) {
      return res.status(404).json({ error: 'Task not found' })
    }
    return res.json({ jsonrpc: '2.0', id: body.id, result: task })
  }

  if (body.method === 'tasks/cancel') {
    const taskId = body.params?.id
    if (!taskId) {
      return res.status(400).json({ error: 'Missing task id' })
    }
    const task = loadTask(taskId)
    if (task) {
      task.status = { state: 'canceled' }
      saveTask(task)
    }
    return res.json({ jsonrpc: '2.0', id: body.id, result: { id: taskId } })
  }

  return res.status(400).json({ error: `Unknown method: ${body.method}` })
})

app.listen(PORT, () => {
  console.log(`claude-code-server listening on :${PORT}`)
  console.log(`Agent: ${loadAgentCard().name}`)
  console.log(`Workspace: ${WORKSPACE}`)
  if (MAX_TURNS !== undefined) {
    console.log(`Max turns: ${MAX_TURNS}`)
  }
})
