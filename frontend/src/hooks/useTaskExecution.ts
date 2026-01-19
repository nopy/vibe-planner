import { useEffect, useRef, useState, useCallback } from 'react'

interface ExecutionEvent {
  type: 'output' | 'error' | 'status' | 'done'
  data: string
  timestamp: string
}

interface UseTaskExecutionReturn {
  output: ExecutionEvent[]
  isStreaming: boolean
  error: string | null
  startStreaming: () => void
  stopStreaming: () => void
}

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8090'

export function useTaskExecution(
  projectId: string,
  taskId: string,
  sessionId: string | null
): UseTaskExecutionReturn {
  const [output, setOutput] = useState<ExecutionEvent[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const eventSourceRef = useRef<EventSource | null>(null)
  const shouldStream = useRef(false)

  const startStreaming = useCallback(() => {
    if (!sessionId || isStreaming || eventSourceRef.current) {
      return
    }

    shouldStream.current = true
    setError(null)

    try {
      const url = `${API_BASE_URL}/api/projects/${projectId}/tasks/${taskId}/output?session_id=${sessionId}`
      const es = new EventSource(url)

      es.onopen = () => {
        console.log(`[useTaskExecution] SSE connected to ${url}`)
        setIsStreaming(true)
      }

      es.addEventListener('output', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          setOutput(prev => [...prev, { type: 'output', data: data.line, timestamp: data.timestamp }])
        } catch (err) {
          console.error('[useTaskExecution] Failed to parse output event:', err)
        }
      })

      es.addEventListener('error', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          setOutput(prev => [...prev, { type: 'error', data: data.message, timestamp: data.timestamp }])
        } catch (err) {
          console.error('[useTaskExecution] Failed to parse error event:', err)
        }
      })

      es.addEventListener('status', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          setOutput(prev => [
            ...prev,
            { type: 'status', data: `Status: ${data.status}`, timestamp: data.timestamp },
          ])
        } catch (err) {
          console.error('[useTaskExecution] Failed to parse status event:', err)
        }
      })

      es.addEventListener('done', (event: MessageEvent) => {
        try {
          const data = JSON.parse(event.data)
          setOutput(prev => [
            ...prev,
            { type: 'done', data: `Execution completed: ${data.reason}`, timestamp: data.timestamp },
          ])
          es.close()
          eventSourceRef.current = null
          setIsStreaming(false)
        } catch (err) {
          console.error('[useTaskExecution] Failed to parse done event:', err)
        }
      })

      es.onerror = event => {
        console.error('[useTaskExecution] SSE error:', event)
        setError('Connection lost. Retrying...')

        // EventSource auto-reconnects, but if it fails permanently we need to handle it
        if (es.readyState === EventSource.CLOSED) {
          setIsStreaming(false)
          eventSourceRef.current = null
        }
      }

      eventSourceRef.current = es
    } catch (err) {
      console.error('[useTaskExecution] Failed to create EventSource:', err)
      setError('Failed to start streaming execution output')
      setIsStreaming(false)
    }
  }, [projectId, taskId, sessionId, isStreaming])

  const stopStreaming = useCallback(() => {
    shouldStream.current = false

    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }

    setIsStreaming(false)
  }, [])

  // Auto-start streaming when sessionId is available
  useEffect(() => {
    if (sessionId && !isStreaming) {
      startStreaming()
    }

    return () => {
      stopStreaming()
    }
  }, [sessionId, isStreaming, startStreaming, stopStreaming])

  return {
    output,
    isStreaming,
    error,
    startStreaming,
    stopStreaming,
  }
}
