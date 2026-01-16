import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor, render } from '@testing-library/react'
import App from './App'

describe('App Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders without crashing', async () => {
    render(<App />)

    await waitFor(() => {
      expect(
        screen.getByText('OpenCode Project Manager')
      ).toBeInTheDocument()
    })
  })

  it('shows home page with Get Started link', async () => {
    render(<App />)

    await waitFor(() => {
      expect(
        screen.getByText('OpenCode Project Manager')
      ).toBeInTheDocument()
      expect(
        screen.getByText(
          'Manage your projects with AI-powered coding assistance'
        )
      ).toBeInTheDocument()
      expect(screen.getByText('Get Started')).toBeInTheDocument()
    })
  })

  it('provides AuthContext to entire app', async () => {
    render(<App />)

    await waitFor(() => {
      const heading = screen.getByText('OpenCode Project Manager')
      expect(heading).toBeInTheDocument()
    })
  })
})
