import { useEffect, useRef, useState, useCallback } from 'react'

import type { Interaction, InteractionMessage, WebSocketMessageType } from '@/types'

interface UseInteractionsReturn {
  messages: Interaction[]
  isConnected: boolean
  isTyping: boolean
  error: string | null
  sendMessage: (content: string) => void
  reconnect: () => void
}

const WS_BASE_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8090/api'
const MAX_RECONNECT_ATTEMPTS = 5
const INITIAL_RECONNECT_DELAY = 1000 // 1 second
const MAX_RECONNECT_DELAY = 16000 // 16 seconds

export function useInteractions(projectId: string, taskId: string): UseInteractionsReturn {
  const [messages, setMessages] = useState<Interaction[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [isTyping, setIsTyping] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const shouldConnectRef = useRef(true)
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const connect = useCallback(() => {
    // Don't reconnect if already connected or connecting
    if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
      return
    }

    const token = localStorage.getItem('token')
    if (!token) {
      setError('Not authenticated')
      setIsConnected(false)
      return
    }

    try {

      const url = `${WS_BASE_URL}/projects/${projectId}/tasks/${taskId}/interact`
      console.log(`[useInteractions] Connecting to ${url}`)

      const ws = new WebSocket(url)

      ws.onopen = () => {
        console.log('[useInteractions] WebSocket connected')
        setIsConnected(true)
        setError(null)
        reconnectAttemptsRef.current = 0

        // Send authentication token
        ws.send(
          JSON.stringify({
            type: 'auth',
            token,
          })
        )
      }

      ws.onmessage = event => {
        try {
          const message: {
            type: WebSocketMessageType
            content?: string
            messages?: Interaction[]
            timestamp: string
          } = JSON.parse(event.data)

          console.log('[useInteractions] Received message:', message.type)

          switch (message.type) {
            case 'history':
              // Initial history load on connect
              if (message.messages) {
                setMessages(message.messages)
              }
              break

            case 'user_message':
            case 'agent_response':
            case 'system_notification':
              // New message received
              if (message.content) {
                const newMessage: Interaction = {
                  id: '', // Backend assigns ID
                  task_id: taskId,
                  user_id: '', // Backend assigns user_id
                  message_type: message.type as 'user_message' | 'agent_response' | 'system_notification',
                  content: message.content,
                  created_at: message.timestamp,
                }
                setMessages(prev => [...prev, newMessage])
              }

              // If agent response, clear typing indicator
              if (message.type === 'agent_response') {
                setIsTyping(false)
                if (typingTimeoutRef.current) {
                  clearTimeout(typingTimeoutRef.current)
                }
              }
              break

            case 'status_update':
              // Agent is thinking/typing
              setIsTyping(true)

              // Auto-hide typing indicator after 30 seconds if no response
              if (typingTimeoutRef.current) {
                clearTimeout(typingTimeoutRef.current)
              }
              typingTimeoutRef.current = setTimeout(() => {
                setIsTyping(false)
              }, 30000)
              break

            case 'error':
              console.error('[useInteractions] Server error:', message.content)
              setError(message.content || 'Unknown error')
              break
          }
        } catch (err) {
          console.error('[useInteractions] Failed to parse message:', err)
        }
      }

      ws.onerror = event => {
        console.error('[useInteractions] WebSocket error:', event)
        setError('Connection error')
      }

      ws.onclose = event => {
        console.log(`[useInteractions] WebSocket closed (code: ${event.code})`)
        setIsConnected(false)
        wsRef.current = null

        // Attempt to reconnect with exponential backoff
        if (shouldConnectRef.current && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          const delay = Math.min(
            INITIAL_RECONNECT_DELAY * Math.pow(2, reconnectAttemptsRef.current),
            MAX_RECONNECT_DELAY
          )

          console.log(
            `[useInteractions] Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current + 1}/${MAX_RECONNECT_ATTEMPTS})`
          )

          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++
            connect()
          }, delay)
        } else if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
          setError('Failed to connect after multiple attempts')
        }
      }

      wsRef.current = ws
    } catch (err) {
      console.error('[useInteractions] Failed to create WebSocket:', err)
      setError('Failed to establish connection')
    }
  }, [projectId, taskId])

  const disconnect = useCallback(() => {
    shouldConnectRef.current = false

    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current)
      typingTimeoutRef.current = null
    }

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setIsConnected(false)
  }, [])

  const sendMessage = useCallback(
    (content: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        setError('Not connected')
        return
      }

      if (!content.trim()) {
        return
      }

      if (content.length > 2000) {
        setError('Message too long (max 2000 characters)')
        return
      }

      try {
        const message: InteractionMessage = {
          type: 'user_message',
          content: content.trim(),
          metadata: {},
        }

        wsRef.current.send(JSON.stringify(message))
        setError(null)
      } catch (err) {
        console.error('[useInteractions] Failed to send message:', err)
        setError('Failed to send message')
      }
    },
    []
  )

  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0
    shouldConnectRef.current = true
    setError(null)
    connect()
  }, [connect])

  // Auto-connect on mount
  useEffect(() => {
    shouldConnectRef.current = true
    
    // Small delay to allow test setup
    const timer = setTimeout(() => {
      connect()
    }, 0)

    return () => {
      clearTimeout(timer)
      disconnect()
    }
  }, [connect, disconnect])

  return {
    messages,
    isConnected,
    isTyping,
    error,
    sendMessage,
    reconnect,
  }
}
