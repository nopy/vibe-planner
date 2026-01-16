import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { createElement } from 'react'
import { useAuth } from './useAuth'
import { TestProviders, createMockUser } from '@/tests/test-utils'

describe('useAuth Hook', () => {
  it('throws error when used outside AuthProvider', () => {
    expect(() => {
      renderHook(() => useAuth())
    }).toThrow('useAuth must be used within an AuthProvider')
  })

  it('returns context value when used within AuthProvider', () => {
    const mockUser = createMockUser()
    const authValue = {
      user: mockUser,
      token: 'test-token',
      isAuthenticated: true,
      isLoading: false,
    }

    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) =>
        createElement(TestProviders, { authValue }, children),
    })

    expect(result.current.user).toEqual(mockUser)
    expect(result.current.token).toBe('test-token')
    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.isLoading).toBe(false)
  })

  it('returns null user when not authenticated', () => {
    const authValue = {
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
    }

    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) =>
        createElement(TestProviders, { authValue }, children),
    })

    expect(result.current.user).toBeNull()
    expect(result.current.token).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
  })

  it('provides login, handleCallback, and logout functions', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => createElement(TestProviders, {}, children),
    })

    expect(typeof result.current.login).toBe('function')
    expect(typeof result.current.handleCallback).toBe('function')
    expect(typeof result.current.logout).toBe('function')
  })
})
