import { useEffect, useRef, useState } from 'react'

import type { PodStatus } from '@/types'

interface UseProjectStatusReturn {
  status: PodStatus | null
  isConnected: boolean
  error: string | null
  reconnect: () => void
}

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8090/api/projects'
const RECONNECT_DELAY = 3000
const MAX_RECONNECT_ATTEMPTS = 5

export function useProjectStatus(projectId: string): UseProjectStatusReturn {
  const [status, setStatus] = useState<PodStatus | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const shouldConnect = useRef(true)

  const connect = () => {
    if (!shouldConnect.current || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      const url = `${WS_BASE_URL}/${projectId}/status`
      const ws = new WebSocket(url)

      ws.onopen = () => {
        console.log(`[useProjectStatus] Connected to ${url}`)
        setIsConnected(true)
        setError(null)
        reconnectAttempts.current = 0
      }

      ws.onmessage = event => {
        try {
          const data = JSON.parse(event.data)

          if (data.status) {
            setStatus(data.status as PodStatus)
          }
        } catch (err) {
          console.error('[useProjectStatus] Failed to parse message:', err)
        }
      }

      ws.onerror = event => {
        console.error('[useProjectStatus] WebSocket error:', event)
        setError('WebSocket connection error')
      }

      ws.onclose = event => {
        console.log(
          `[useProjectStatus] Connection closed: code=${event.code}, reason=${event.reason}`
        )
        setIsConnected(false)
        wsRef.current = null

        if (shouldConnect.current && reconnectAttempts.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts.current++
          console.log(
            `[useProjectStatus] Reconnecting in ${RECONNECT_DELAY}ms (attempt ${reconnectAttempts.current}/${MAX_RECONNECT_ATTEMPTS})`
          )

          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect()
          }, RECONNECT_DELAY)
        } else if (reconnectAttempts.current >= MAX_RECONNECT_ATTEMPTS) {
          setError('Maximum reconnection attempts reached. Please refresh the page.')
        }
      }

      wsRef.current = ws
    } catch (err) {
      console.error('[useProjectStatus] Failed to create WebSocket:', err)
      setError('Failed to create WebSocket connection')
    }
  }

  const disconnect = () => {
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
  }

  const reconnect = () => {
    disconnect()
    reconnectAttempts.current = 0
    setError(null)
    shouldConnect.current = true
    connect()
  }

  useEffect(() => {
    shouldConnect.current = true
    connect()

    return () => {
      disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId])

  return {
    status,
    isConnected,
    error,
    reconnect,
  }
}
