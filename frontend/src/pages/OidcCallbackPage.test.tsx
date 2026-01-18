import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { OidcCallbackPage } from './OidcCallbackPage'
import { AuthContext } from '@/contexts/AuthContext'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

describe('OidcCallbackPage', () => {
  const mockHandleCallback = vi.fn()
  const defaultAuthValue = {
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    login: async () => {},
    handleCallback: mockHandleCallback,
    logout: () => {},
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('displays loading state initially', () => {
    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?code=test-code']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    expect(screen.getByText('Completing authentication...')).toBeInTheDocument()
  })

  it('calls handleCallback with authorization code', async () => {
    mockHandleCallback.mockResolvedValue(undefined)

    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?code=auth-code-123']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(mockHandleCallback).toHaveBeenCalledWith('auth-code-123')
    })
  })

  it('shows error when error parameter is present', async () => {
    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?error=access_denied']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(screen.getByText('Authentication Failed')).toBeInTheDocument()
      expect(screen.getByText('Authentication error: access_denied')).toBeInTheDocument()
    })

    expect(mockHandleCallback).not.toHaveBeenCalled()
  })

  it('shows error when no code is provided', async () => {
    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(screen.getByText('Authentication Failed')).toBeInTheDocument()
      expect(screen.getByText('No authorization code received')).toBeInTheDocument()
    })

    expect(mockHandleCallback).not.toHaveBeenCalled()
  })

  it('shows error when handleCallback fails', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    mockHandleCallback.mockRejectedValue(new Error('Callback failed'))

    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?code=bad-code']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(screen.getByText('Authentication Failed')).toBeInTheDocument()
      expect(screen.getByText('Failed to complete authentication')).toBeInTheDocument()
    })

    expect(mockHandleCallback).toHaveBeenCalledWith('bad-code')
    expect(consoleErrorSpy).toHaveBeenCalled()

    consoleErrorSpy.mockRestore()
  })

  it('navigates to /projects on successful callback', async () => {
    mockHandleCallback.mockResolvedValue(undefined)

    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?code=success-code']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(mockHandleCallback).toHaveBeenCalledWith('success-code')
      expect(mockNavigate).toHaveBeenCalledWith('/projects')
    })
  })

  it('prevents duplicate processing with hasProcessed ref', async () => {
    mockHandleCallback.mockResolvedValue(undefined)

    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?code=duplicate-code']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(mockHandleCallback).toHaveBeenCalledTimes(1)
    })
  })

  it('renders "Try Again" button on error', async () => {
    render(
      <AuthContext.Provider value={defaultAuthValue}>
        <MemoryRouter initialEntries={['/?error=invalid_request']}>
          <Routes>
            <Route path="/" element={<OidcCallbackPage />} />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    await waitFor(() => {
      expect(screen.getByText('Try Again')).toBeInTheDocument()
    })

    const tryAgainButton = screen.getByRole('button', { name: 'Try Again' })
    expect(tryAgainButton).toHaveClass('bg-blue-600')
  })
})
