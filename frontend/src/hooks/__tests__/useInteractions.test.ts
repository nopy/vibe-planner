import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

import { useInteractions } from '../useInteractions'

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.CONNECTING
  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null

  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code: 1000 }))
    }
  })

  // Helper to simulate connection
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN
    if (this.onopen) {
      this.onopen(new Event('open'))
    }
  }

  // Helper to simulate message
  simulateMessage(data: unknown) {
    if (this.onmessage) {
      this.onmessage(new MessageEvent('message', { data: JSON.stringify(data) }))
    }
  }

  // Helper to simulate error
  simulateError() {
    if (this.onerror) {
      this.onerror(new Event('error'))
    }
  }

  // Helper to simulate close
  simulateClose(code = 1000) {
    this.readyState = MockWebSocket.CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close', { code }))
    }
  }
}

describe('useInteractions', () => {
  let mockWs: MockWebSocket

  beforeEach(() => {
    vi.useFakeTimers()
    mockWs = new MockWebSocket()
    global.WebSocket = vi.fn(() => mockWs) as unknown as typeof WebSocket
    localStorage.setItem('token', 'test-token')
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('should initialize with default state', () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    expect(result.current.messages).toEqual([])
    expect(result.current.isConnected).toBe(false)
    expect(result.current.isTyping).toBe(false)
    expect(result.current.error).toBe(null)
  })

  it('should connect to WebSocket on mount', () => {
    renderHook(() => useInteractions('proj-1', 'task-1'))

    expect(global.WebSocket).toHaveBeenCalledWith(
      'ws://localhost:8090/api/projects/proj-1/tasks/task-1/interact'
    )
  })

  it('should set error if not authenticated', () => {
    localStorage.removeItem('token')

    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    expect(result.current.error).toBe('Not authenticated')
  })

  it('should set isConnected to true on WebSocket open', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })
  })

  it('should send auth token on connection', async () => {
    renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(mockWs.send).toHaveBeenCalledWith(
        JSON.stringify({
          type: 'auth',
          token: 'test-token',
        })
      )
    })
  })

  it('should load history messages on connect', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateMessage({
        type: 'history',
        messages: [
          {
            id: '1',
            task_id: 'task-1',
            user_id: 'user-1',
            message_type: 'user_message',
            content: 'Hello',
            created_at: '2026-01-19T19:00:00Z',
          },
          {
            id: '2',
            task_id: 'task-1',
            user_id: 'user-1',
            message_type: 'agent_response',
            content: 'Hi there!',
            created_at: '2026-01-19T19:00:01Z',
          },
        ],
        timestamp: '2026-01-19T19:00:02Z',
      })
    })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(2)
      expect(result.current.messages[0].content).toBe('Hello')
      expect(result.current.messages[1].content).toBe('Hi there!')
    })
  })

  it('should append new user message to messages array', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateMessage({
        type: 'user_message',
        content: 'New message',
        timestamp: '2026-01-19T19:00:00Z',
      })
    })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0].content).toBe('New message')
      expect(result.current.messages[0].message_type).toBe('user_message')
    })
  })

  it('should append new agent response and clear typing indicator', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    // Set typing indicator
    act(() => {
      mockWs.simulateMessage({
        type: 'status_update',
        timestamp: '2026-01-19T19:00:00Z',
      })
    })

    await waitFor(() => {
      expect(result.current.isTyping).toBe(true)
    })

    // Agent responds
    act(() => {
      mockWs.simulateMessage({
        type: 'agent_response',
        content: 'Agent reply',
        timestamp: '2026-01-19T19:00:01Z',
      })
    })

    await waitFor(() => {
      expect(result.current.messages).toHaveLength(1)
      expect(result.current.messages[0].content).toBe('Agent reply')
      expect(result.current.isTyping).toBe(false)
    })
  })

  it('should show typing indicator on status_update', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateMessage({
        type: 'status_update',
        timestamp: '2026-01-19T19:00:00Z',
      })
    })

    await waitFor(() => {
      expect(result.current.isTyping).toBe(true)
    })
  })

  it('should auto-hide typing indicator after 30 seconds', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateMessage({
        type: 'status_update',
        timestamp: '2026-01-19T19:00:00Z',
      })
    })

    await waitFor(() => {
      expect(result.current.isTyping).toBe(true)
    })

    act(() => {
      vi.advanceTimersByTime(30000) // 30 seconds
    })

    await waitFor(() => {
      expect(result.current.isTyping).toBe(false)
    })
  })

  it('should set error on server error message', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateMessage({
        type: 'error',
        content: 'Server error occurred',
        timestamp: '2026-01-19T19:00:00Z',
      })
    })

    await waitFor(() => {
      expect(result.current.error).toBe('Server error occurred')
    })
  })

  it('should send message via WebSocket', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    act(() => {
      result.current.sendMessage('Hello, AI!')
    })

    expect(mockWs.send).toHaveBeenCalledWith(
      JSON.stringify({
        type: 'user_message',
        content: 'Hello, AI!',
        metadata: {},
      })
    )
  })

  it('should reject empty messages', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    const sendCallCount = mockWs.send.mock.calls.length

    act(() => {
      result.current.sendMessage('   ')
    })

    // Should not call send again
    expect(mockWs.send).toHaveBeenCalledTimes(sendCallCount)
  })

  it('should reject messages longer than 2000 characters', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    act(() => {
      result.current.sendMessage('a'.repeat(2001))
    })

    await waitFor(() => {
      expect(result.current.error).toBe('Message too long (max 2000 characters)')
    })
  })

  it('should set error when sending message while disconnected', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      result.current.sendMessage('Hello')
    })

    await waitFor(() => {
      expect(result.current.error).toBe('Not connected')
    })
  })

  it('should attempt reconnection with exponential backoff', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(true)
    })

    // Simulate disconnect
    act(() => {
      mockWs.simulateClose(1006) // Abnormal closure
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    })

    // First reconnect attempt after 1 second
    act(() => {
      vi.advanceTimersByTime(1000)
    })

    expect(global.WebSocket).toHaveBeenCalledTimes(2)
  })

  it('should stop reconnecting after MAX_RECONNECT_ATTEMPTS', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    // Simulate 5 consecutive failures
    for (let i = 0; i < 5; i++) {
      act(() => {
        mockWs.simulateClose(1006)
      })

      act(() => {
        vi.advanceTimersByTime(20000) // Max delay
      })

      mockWs = new MockWebSocket() // Reset for next attempt
      global.WebSocket = vi.fn(() => mockWs) as unknown as typeof WebSocket
    }

    await waitFor(() => {
      expect(result.current.error).toBe('Failed to connect after multiple attempts')
    })
  })

  it('should reconnect manually via reconnect()', async () => {
    const { result } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    act(() => {
      mockWs.simulateClose()
    })

    await waitFor(() => {
      expect(result.current.isConnected).toBe(false)
    })

    mockWs = new MockWebSocket()
    global.WebSocket = vi.fn(() => mockWs) as unknown as typeof WebSocket

    act(() => {
      result.current.reconnect()
    })

    expect(global.WebSocket).toHaveBeenCalled()
    expect(result.current.error).toBe(null)
  })

  it('should close WebSocket on unmount', () => {
    const { unmount } = renderHook(() => useInteractions('proj-1', 'task-1'))

    act(() => {
      mockWs.simulateOpen()
    })

    unmount()

    expect(mockWs.close).toHaveBeenCalled()
  })
})
