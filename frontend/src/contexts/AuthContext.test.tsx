import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { AuthProvider } from './AuthContext'
import { useAuth } from '@/hooks/useAuth'
import * as apiModule from '@/services/api'

vi.mock('@/services/api', () => ({
  api: {
    get: vi.fn(),
    interceptors: {
      request: { use: vi.fn(), handlers: [] },
      response: { use: vi.fn(), handlers: [] },
    },
    defaults: { baseURL: 'http://localhost:8090/api' },
  },
}))

describe('AuthContext', () => {
  const mockApi = vi.mocked(apiModule.api)

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mockApi.get.mockResolvedValue({ data: null })
  })

  describe('Initialization', () => {
    it('initializes with no user when no token in localStorage', async () => {
      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.user).toBeNull()
      expect(result.current.token).toBeNull()
      expect(result.current.isAuthenticated).toBe(false)
    })
  })

  describe('logout()', () => {
    it('clears localStorage and state', async () => {
      localStorage.setItem('token', 'test-token')

      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      act(() => {
        result.current.logout()
      })

      expect(localStorage.getItem('token')).toBeNull()
      expect(result.current.user).toBeNull()
      expect(result.current.token).toBeNull()
      expect(result.current.isAuthenticated).toBe(false)
    })
  })

  describe('isAuthenticated computed property', () => {
    it('is false when user and token are null', async () => {
      const { result } = renderHook(() => useAuth(), {
        wrapper: AuthProvider,
      })

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false)
      })

      expect(result.current.isAuthenticated).toBe(false)
    })
  })
})
