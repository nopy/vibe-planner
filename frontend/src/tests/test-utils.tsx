import { ReactElement, ReactNode } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { BrowserRouter, MemoryRouter } from 'react-router-dom'
import { AuthContext } from '@/contexts/AuthContext'

interface User {
  id: string
  oidc_subject: string
  email: string
  name: string
  picture_url: string
  last_login_at: string | null
  created_at: string
  updated_at: string
}

interface AuthContextValue {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: () => Promise<void>
  handleCallback: (code: string) => Promise<void>
  logout: () => void
}

interface TestProvidersProps {
  children: ReactNode
  authValue?: Partial<AuthContextValue>
  initialRoute?: string
}

const defaultAuthValue: AuthContextValue = {
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: false,
  login: async () => {},
  handleCallback: async () => {},
  logout: () => {},
}

export function TestProviders({ children, authValue, initialRoute = '/' }: TestProvidersProps) {
  const mergedAuthValue = { ...defaultAuthValue, ...authValue }

  return (
    <AuthContext.Provider value={mergedAuthValue}>
      <MemoryRouter initialEntries={[initialRoute]}>{children}</MemoryRouter>
    </AuthContext.Provider>
  )
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  authValue?: Partial<AuthContextValue>
  initialRoute?: string
  useMemoryRouter?: boolean
}

// eslint-disable-next-line react-refresh/only-export-components
export function renderWithProviders(
  ui: ReactElement,
  {
    authValue,
    initialRoute = '/',
    useMemoryRouter = true,
    ...renderOptions
  }: CustomRenderOptions = {}
) {
  const Wrapper = ({ children }: { children: ReactNode }) => {
    const mergedAuthValue = { ...defaultAuthValue, ...authValue }

    if (useMemoryRouter) {
      return (
        <AuthContext.Provider value={mergedAuthValue}>
          <MemoryRouter initialEntries={[initialRoute]}>{children}</MemoryRouter>
        </AuthContext.Provider>
      )
    }

    return (
      <AuthContext.Provider value={mergedAuthValue}>
        <BrowserRouter>{children}</BrowserRouter>
      </AuthContext.Provider>
    )
  }

  return render(ui, { wrapper: Wrapper, ...renderOptions })
}

// eslint-disable-next-line react-refresh/only-export-components
export function createMockUser(overrides?: Partial<User>): User {
  return {
    id: 'test-user-id',
    oidc_subject: 'test-subject',
    email: 'test@example.com',
    name: 'Test User',
    picture_url: 'https://example.com/avatar.jpg',
    last_login_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

// eslint-disable-next-line react-refresh/only-export-components
export * from '@testing-library/react'
// eslint-disable-next-line react-refresh/only-export-components
export { renderWithProviders as render }
