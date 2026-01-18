import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from './ProtectedRoute'
import { AuthContext } from '@/contexts/AuthContext'

const defaultAuthValue = {
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: false,
  login: async () => {},
  handleCallback: async () => {},
  logout: () => {},
}

describe('ProtectedRoute', () => {
  it('shows loading state when isLoading is true', () => {
    render(
      <AuthContext.Provider value={{ ...defaultAuthValue, isLoading: true }}>
        <MemoryRouter>
          <ProtectedRoute>
            <div>Protected Content</div>
          </ProtectedRoute>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    expect(screen.getByText('Loading...')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('renders children when authenticated', () => {
    render(
      <AuthContext.Provider value={{ ...defaultAuthValue, isAuthenticated: true }}>
        <MemoryRouter>
          <ProtectedRoute>
            <div>Protected Content</div>
          </ProtectedRoute>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    expect(screen.getByText('Protected Content')).toBeInTheDocument()
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
  })

  it('redirects to /login when not authenticated', () => {
    render(
      <AuthContext.Provider value={{ ...defaultAuthValue, isAuthenticated: false }}>
        <MemoryRouter initialEntries={['/protected']}>
          <Routes>
            <Route
              path="/protected"
              element={
                <ProtectedRoute>
                  <div>Protected Content</div>
                </ProtectedRoute>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    expect(screen.getByText('Login Page')).toBeInTheDocument()
  })

  it('preserves location state for redirect', () => {
    render(
      <AuthContext.Provider value={{ ...defaultAuthValue, isAuthenticated: false }}>
        <MemoryRouter initialEntries={['/protected/resource']}>
          <Routes>
            <Route
              path="/protected/resource"
              element={
                <ProtectedRoute>
                  <div>Protected Content</div>
                </ProtectedRoute>
              }
            />
            <Route path="/login" element={<div>Login Page</div>} />
          </Routes>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
    expect(screen.getByText('Login Page')).toBeInTheDocument()
  })

  it('shows spinner with correct classes during loading', () => {
    render(
      <AuthContext.Provider value={{ ...defaultAuthValue, isLoading: true }}>
        <MemoryRouter>
          <ProtectedRoute>
            <div>Protected Content</div>
          </ProtectedRoute>
        </MemoryRouter>
      </AuthContext.Provider>
    )

    const spinner = screen.getByText('Loading...').previousElementSibling as HTMLElement
    expect(spinner).toHaveClass('animate-spin')
    expect(spinner).toHaveClass('border-blue-600')
  })
})
