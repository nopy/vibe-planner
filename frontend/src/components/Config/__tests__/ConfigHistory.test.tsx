import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ConfigHistory } from '../ConfigHistory'
import * as api from '@/services/api'
import { buildConfigHistory, buildConfig } from '@/tests/factories/opencodeConfig'

vi.mock('@/services/api', () => ({
  getConfigHistory: vi.fn(),
}))

describe('ConfigHistory', () => {
  const projectId = 'test-project-123'
  const mockOnRollback = vi.fn().mockResolvedValue(undefined)

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loading state shows spinner', () => {
    vi.mocked(api.getConfigHistory).mockImplementation(
      () => new Promise(() => {})
    )

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    const spinner = document.querySelector('.animate-spin')
    expect(spinner).toBeTruthy()
    expect(spinner).toHaveClass('animate-spin')
  })

  it('error state shows error message', async () => {
    vi.mocked(api.getConfigHistory).mockRejectedValue(new Error('Network error'))

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText(/failed to load version history/i)).toBeInTheDocument()
    })
  })

  it('empty history shows "no versions" message', async () => {
    vi.mocked(api.getConfigHistory).mockResolvedValue([])

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText(/no configuration history found/i)).toBeInTheDocument()
    })
  })

  it('renders version list with active version highlighted', async () => {
    const history = buildConfigHistory(3)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v3')).toBeInTheDocument()
    })

    const activeVersion = screen.getByText('Active')
    expect(activeVersion).toBeInTheDocument()

    expect(screen.getByText('v1')).toBeInTheDocument()
    expect(screen.getByText('v2')).toBeInTheDocument()
  })

  it('expand/collapse details works', async () => {
    const user = userEvent.setup()
    const history = buildConfigHistory(2)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v1')).toBeInTheDocument()
    })

    const expandButtons = screen.getAllByRole('button', { name: '' })
    const firstExpandButton = expandButtons.find((btn) => btn.querySelector('svg'))

    if (!firstExpandButton) throw new Error('Expand button not found')

    await user.click(firstExpandButton)

    expect(screen.getByText(/temperature/i)).toBeInTheDocument()
    expect(screen.getByText(/max tokens/i)).toBeInTheDocument()

    await user.click(firstExpandButton)

    await waitFor(() => {
      expect(screen.queryByText(/temperature/i)).not.toBeInTheDocument()
    })
  })

  it('rollback button hidden for active version', async () => {
    const history = buildConfigHistory(3)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('Active')).toBeInTheDocument()
    })

    const rollbackButtons = screen.queryAllByRole('button', { name: /rollback/i })

    expect(rollbackButtons).toHaveLength(2)
  })

  it('rollback with confirm=true calls onRollback', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const history = buildConfigHistory(3)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v1')).toBeInTheDocument()
    })

    const rollbackButtons = screen.getAllByRole('button', { name: /rollback/i })

    await user.click(rollbackButtons[0])

    expect(confirmSpy).toHaveBeenCalled()
    expect(mockOnRollback).toHaveBeenCalledWith(1)

    confirmSpy.mockRestore()
  })

  it('rollback with confirm=false does not call onRollback', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const history = buildConfigHistory(3)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v1')).toBeInTheDocument()
    })

    const rollbackButtons = screen.getAllByRole('button', { name: /rollback/i })

    await user.click(rollbackButtons[0])

    expect(confirmSpy).toHaveBeenCalled()
    expect(mockOnRollback).not.toHaveBeenCalled()

    confirmSpy.mockRestore()
  })

  it('rolling back disables buttons (isRollingBack state)', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const history = buildConfigHistory(3)
    vi.mocked(api.getConfigHistory).mockResolvedValue(history)

    let resolveRollback: () => void
    const rollbackPromise = new Promise<void>((resolve) => {
      resolveRollback = resolve
    })
    mockOnRollback.mockReturnValue(rollbackPromise)

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v1')).toBeInTheDocument()
    })

    const rollbackButtons = screen.getAllByRole('button', { name: /rollback/i })

    await user.click(rollbackButtons[0])

    await waitFor(() => {
      rollbackButtons.forEach((btn) => {
        expect(btn).toBeDisabled()
      })
    })

    resolveRollback!()

    confirmSpy.mockRestore()
  })

  it('expanded details show temperature, tokens, tools, created_by', async () => {
    const user = userEvent.setup()
    const config = buildConfig({
      temperature: 0.8,
      max_tokens: 5000,
      enabled_tools: ['file_ops', 'web_search'],
      created_by: 'admin@example.com',
    })
    vi.mocked(api.getConfigHistory).mockResolvedValue([config])

    render(<ConfigHistory projectId={projectId} onRollback={mockOnRollback} />)

    await waitFor(() => {
      expect(screen.getByText('v1')).toBeInTheDocument()
    })

    const expandButtons = screen.getAllByRole('button', { name: '' })
    const expandButton = expandButtons.find((btn) => btn.querySelector('svg'))

    if (!expandButton) throw new Error('Expand button not found')

    await user.click(expandButton)

    expect(screen.getByText('0.8')).toBeInTheDocument()
    expect(screen.getByText('5000')).toBeInTheDocument()
    expect(screen.getByText('file_ops')).toBeInTheDocument()
    expect(screen.getByText('web_search')).toBeInTheDocument()
    expect(screen.getByText('admin@example.com')).toBeInTheDocument()
  })
})
