import { useEffect, useRef, useState, useCallback } from 'react'

import type { FileChangeEvent } from '@/types'

interface UseFileWatchReturn {
  events: FileChangeEvent[]
  isConnected: boolean
  error: string | null
  reconnect: () => void
  clearEvents: () => void
}

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8090/api/projects'
const BASE_DELAY = 1000
const MAX_DELAY = 30000
const MAX_RECONNECT_ATTEMPTS = 10

function getBackoffDelay(attempt: number): number {
  const exp = Math.min(MAX_DELAY, BASE_DELAY * 2 ** Math.max(0, attempt - 1))
  return Math.floor(Math.random() * exp)
}

export function useFileWatch(projectId: string): UseFileWatchReturn {
  const [events, setEvents] = useState<FileChangeEvent[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const shouldConnect = useRef(true)
  const lastVersion = useRef<number>(0)

  const applyEvent = useCallback((event: FileChangeEvent) => {
    if (event.version && event.version <= lastVersion.current) {
      console.log(
        `[useFileWatch] Ignoring stale event (version ${event.version} <= ${lastVersion.current})`
      )
      return
    }

    if (event.version) {
      lastVersion.current = event.version
    }

    setEvents(prev => {
      const MAX_EVENTS = 100
      const next = [...prev, event]
      if (next.length > MAX_EVENTS) {
        return next.slice(-MAX_EVENTS)
      }
      return next
    })

    console.log(`[useFileWatch] File ${event.type}: ${event.path}`)
  }, [])

  const connect = useCallback(() => {
    if (!shouldConnect.current || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      const url = `${WS_BASE_URL}/${projectId}/files/watch`
      const ws = new WebSocket(url)

      ws.onopen = () => {
        console.log(`[useFileWatch] Connected to ${url}`)
        setIsConnected(true)
        setError(null)
        reconnectAttempts.current = 0
      }

      ws.onmessage = event => {
        try {
          const data = JSON.parse(event.data) as FileChangeEvent
          applyEvent(data)
        } catch (err) {
          console.error('[useFileWatch] Failed to parse message:', err)
        }
      }

      ws.onerror = event => {
        console.error('[useFileWatch] WebSocket error:', event)
        setError('WebSocket connection error')
      }

      ws.onclose = event => {
        console.log(`[useFileWatch] Connection closed: code=${event.code}, reason=${event.reason}`)
        setIsConnected(false)
        wsRef.current = null

        if (shouldConnect.current && reconnectAttempts.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts.current++
          const delay = getBackoffDelay(reconnectAttempts.current)
          console.log(
            `[useFileWatch] Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current}/${MAX_RECONNECT_ATTEMPTS})`
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
      console.error('[useFileWatch] Failed to create WebSocket:', err)
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

  const clearEvents = useCallback(() => {
    setEvents([])
  }, [])

  useEffect(() => {
    shouldConnect.current = true
    connect()

    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return {
    events,
    isConnected,
    error,
    reconnect,
    clearEvents,
  }
}
