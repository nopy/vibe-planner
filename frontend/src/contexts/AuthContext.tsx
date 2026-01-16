import { createContext, useState, useEffect, ReactNode } from 'react'
import { api } from '../services/api'

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

interface AuthContextType {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
  login: () => Promise<void>
  handleCallback: (code: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'))
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const initAuth = async () => {
      const storedToken = localStorage.getItem('token')
      if (storedToken) {
        try {
          const response = await api.get('/auth/me')
          setUser(response.data)
          setToken(storedToken)
        } catch (error) {
          localStorage.removeItem('token')
          setToken(null)
          setUser(null)
        }
      }
      setIsLoading(false)
    }

    initAuth()
  }, [])

  const login = async () => {
    try {
      const response = await api.get('/auth/oidc/login')
      window.location.href = response.data.authorization_url
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    }
  }

  const handleCallback = async (code: string) => {
    try {
      const response = await api.get('/auth/oidc/callback', {
        params: { code },
      })

      const { token: newToken, user: newUser } = response.data
      localStorage.setItem('token', newToken)
      setToken(newToken)
      setUser(newUser)
    } catch (error) {
      console.error('Callback handling failed:', error)
      throw error
    }
  }

  const logout = () => {
    localStorage.removeItem('token')
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isAuthenticated: !!user && !!token,
        isLoading,
        login,
        handleCallback,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}


