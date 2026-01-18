import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LoginPage } from './LoginPage'
import { render } from '@/tests/test-utils'

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

describe('LoginPage', () => {
  beforeEach(() => {
    mockNavigate.mockClear()
  })

  it('renders login page with correct text', () => {
    render(<LoginPage />, {
      authValue: { isAuthenticated: false },
    })

    expect(screen.getByText('OpenCode Project Manager')).toBeInTheDocument()
    expect(
      screen.getByText(/Sign in to manage your projects with AI-powered coding assistance/i)
    ).toBeInTheDocument()
    expect(screen.getByText('Login with Keycloak')).toBeInTheDocument()
  })

  it('calls login function when button is clicked', async () => {
    const user = userEvent.setup()
    const loginMock = vi.fn().mockResolvedValue(undefined)

    render(<LoginPage />, {
      authValue: { isAuthenticated: false, login: loginMock },
    })

    const loginButton = screen.getByRole('button', {
      name: 'Login with Keycloak',
    })

    await user.click(loginButton)

    expect(loginMock).toHaveBeenCalledTimes(1)
  })

  it('handles login errors gracefully', async () => {
    const user = userEvent.setup()
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const loginMock = vi.fn().mockRejectedValue(new Error('Login failed'))

    render(<LoginPage />, {
      authValue: { isAuthenticated: false, login: loginMock },
    })

    const loginButton = screen.getByRole('button', {
      name: 'Login with Keycloak',
    })

    await user.click(loginButton)

    await waitFor(() => {
      expect(loginMock).toHaveBeenCalledTimes(1)
      expect(consoleErrorSpy).toHaveBeenCalledWith('Login failed:', expect.any(Error))
    })

    consoleErrorSpy.mockRestore()
  })

  it('redirects to /projects when already authenticated', async () => {
    render(<LoginPage />, {
      authValue: { isAuthenticated: true },
      initialRoute: '/login',
    })

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/projects')
    })
  })

  it('has correct button styling', () => {
    render(<LoginPage />, {
      authValue: { isAuthenticated: false },
    })

    const button = screen.getByRole('button', { name: 'Login with Keycloak' })

    expect(button).toHaveClass('bg-blue-600')
    expect(button).toHaveClass('text-white')
    expect(button).toHaveClass('rounded-lg')
    expect(button).toHaveClass('hover:bg-blue-700')
  })
})
