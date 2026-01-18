import { useEffect, useRef, useState, useCallback } from 'react'

import type { Task } from '@/types'

interface TaskEvent {
  type: 'snapshot' | 'created' | 'updated' | 'moved' | 'deleted'
  task?: Task
  tasks?: Task[]
  task_id?: string
  version: number
}

interface UseTaskUpdatesReturn {
  tasks: Task[] | null
  isConnected: boolean
  error: string | null
  reconnect: () => void
}

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8090/api/projects'
const BASE_DELAY = 1000
const MAX_DELAY = 30000
const MAX_RECONNECT_ATTEMPTS = 10

function getBackoffDelay(attempt: number): number {
  const exp = Math.min(MAX_DELAY, BASE_DELAY * 2 ** Math.max(0, attempt - 1))
  return Math.floor(Math.random() * exp)
}

export function useTaskUpdates(projectId: string): UseTaskUpdatesReturn {
  const [tasks, setTasks] = useState<Task[] | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const shouldConnect = useRef(true)
  const lastVersion = useRef<number>(0)

  const applyEvent = useCallback((event: TaskEvent) => {
    if (event.version <= lastVersion.current && event.type !== 'snapshot') {
      console.log(`[useTaskUpdates] Ignoring stale event (version ${event.version} <= ${lastVersion.current})`)
      return
    }

    lastVersion.current = event.version

    switch (event.type) {
      case 'snapshot':
        if (event.tasks) {
          setTasks(event.tasks)
          console.log(`[useTaskUpdates] Applied snapshot with ${event.tasks.length} tasks (version ${event.version})`)
        }
        break

      case 'created':
        if (event.task) {
          setTasks(prev => (prev ? [...prev, event.task!] : [event.task!]))
          console.log(`[useTaskUpdates] Task created: ${event.task.id}`)
        }
        break

      case 'updated':
      case 'moved':
        if (event.task) {
          setTasks(prev =>
            prev ? prev.map(t => (t.id === event.task!.id ? event.task! : t)) : [event.task!]
          )
          console.log(`[useTaskUpdates] Task ${event.type}: ${event.task.id}`)
        }
        break

      case 'deleted':
        if (event.task_id) {
          setTasks(prev => (prev ? prev.filter(t => t.id !== event.task_id) : null))
          console.log(`[useTaskUpdates] Task deleted: ${event.task_id}`)
        }
        break
    }
  }, [])

  const connect = useCallback(() => {
    if (!shouldConnect.current || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      const url = `${WS_BASE_URL}/${projectId}/tasks/stream`
      const ws = new WebSocket(url)

      ws.onopen = () => {
        console.log(`[useTaskUpdates] Connected to ${url}`)
        setIsConnected(true)
        setError(null)
        reconnectAttempts.current = 0
      }

      ws.onmessage = event => {
        try {
          const data = JSON.parse(event.data) as TaskEvent
          applyEvent(data)
        } catch (err) {
          console.error('[useTaskUpdates] Failed to parse message:', err)
        }
      }

      ws.onerror = event => {
        console.error('[useTaskUpdates] WebSocket error:', event)
        setError('WebSocket connection error')
      }

      ws.onclose = event => {
        console.log(
          `[useTaskUpdates] Connection closed: code=${event.code}, reason=${event.reason}`
        )
        setIsConnected(false)
        wsRef.current = null

        if (shouldConnect.current && reconnectAttempts.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts.current++
          const delay = getBackoffDelay(reconnectAttempts.current)
          console.log(
            `[useTaskUpdates] Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current}/${MAX_RECONNECT_ATTEMPTS})`
          )

          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, delay)
        } else if (reconnectAttempts.current >= MAX_RECONNECT_ATTEMPTS) {
          setError('Maximum reconnection attempts reached. Please refresh the page.')
        }
      }

      wsRef.current = ws
    } catch (err) {
      console.error('[useTaskUpdates] Failed to create WebSocket:', err)
      setError('Failed to create WebSocket connection')
    }
  }, [projectId, applyEvent])

  const disconnect = useCallback(() => {
    shouldConnect.current = false

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    if (wsRef.current) {
      wsRef.current.close(1000, 'Component unmounted')
      wsRef.current = null
    }

    setIsConnected(false)
  }, [])

  const reconnect = useCallback(() => {
    disconnect()
    reconnectAttempts.current = 0
    setError(null)
    shouldConnect.current = true
    connect()
  }, [disconnect, connect])

  useEffect(() => {
    shouldConnect.current = true
    connect()

    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return {
    tasks,
    isConnected,
    error,
    reconnect,
  }
}
