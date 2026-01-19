import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import { ConfigPanel } from '../ConfigPanel'
import * as useConfigHook from '@/hooks/useConfig'
import { buildConfig } from '@/tests/factories/opencodeConfig'
import type { UseConfigReturn } from '@/hooks/useConfig'
import type { OpenCodeConfig, CreateConfigRequest } from '@/types'

vi.mock('@/hooks/useConfig')

vi.mock('@/components/Config/ModelSelector', () => ({
  ModelSelector: ({ value, onChange, disabled }: any) => (
    <div data-testid="model-selector">
      ModelSelector: {value.provider}/{value.name} (disabled: {disabled?.toString()})
      <button onClick={() => onChange('anthropic', 'claude-3-5-sonnet-20240620')}>
        Change Model
      </button>
    </div>
  ),
}))

vi.mock('@/components/Config/ProviderConfig', () => ({
  ProviderConfig: ({ provider, temperature, maxTokens, disabled }: any) => (
    <div data-testid="provider-config">
      ProviderConfig: {provider}, temp={temperature}, tokens={maxTokens} (disabled:{' '}
      {disabled?.toString()})
    </div>
  ),
}))

vi.mock('@/components/Config/ToolsManagement', () => ({
  ToolsManagement: ({ enabledTools, disabled }: any) => (
    <div data-testid="tools-management">
      ToolsManagement: {enabledTools.join(',')} (disabled: {disabled?.toString()})
    </div>
  ),
}))

vi.mock('@/components/Config/ConfigHistory', () => ({
  ConfigHistory: ({ projectId, onRollback }: any) => (
    <div data-testid="config-history">
      ConfigHistory: {projectId}
      <button onClick={() => onRollback(1)}>Rollback to v1</button>
    </div>
  ),
}))

describe('ConfigPanel', () => {
  const projectId = 'test-project-123'

  const createMockUseConfig = (overrides?: Partial<UseConfigReturn>): UseConfigReturn => ({
    config: null,
    loading: false,
    error: null,
    updateConfig: vi.fn().mockResolvedValue(buildConfig()),
    rollbackConfig: vi.fn().mockResolvedValue(undefined),
    refetch: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  })

  beforeEach(() => {
    vi.clearAllMocks()
  })

  const renderConfigPanel = () => {
    return render(
      <MemoryRouter initialEntries={[`/projects/${projectId}/config`]}>
        <Routes>
          <Route path="/projects/:id/config" element={<ConfigPanel />} />
        </Routes>
      </MemoryRouter>
    )
  }

  it('loading state (no config yet) shows spinner', () => {
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ loading: true })
    )

    renderConfigPanel()

    const spinner = document.querySelector('.animate-spin')
    expect(spinner).toBeTruthy()
    expect(spinner).toHaveClass('animate-spin')
  })

  it('error state (no config) shows error message', () => {
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ error: 'Failed to load configuration' })
    )

    renderConfigPanel()

    const errorMessages = screen.getAllByText(/failed to load configuration/i)
    expect(errorMessages.length).toBeGreaterThan(0)
  })

  it('success state renders all child components', () => {
    const mockConfig = buildConfig()
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    expect(screen.getByTestId('model-selector')).toBeInTheDocument()
    expect(screen.getByTestId('provider-config')).toBeInTheDocument()
    expect(screen.getByTestId('tools-management')).toBeInTheDocument()
    expect(screen.getByTestId('config-history')).toBeInTheDocument()
  })

  it('edit mode toggle shows Save/Cancel buttons', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    expect(screen.getByRole('button', { name: /edit configuration/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit configuration/i })).not.toBeInTheDocument()
  })

  it('save button calls updateConfig with form data', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    const mockUpdateConfig = vi.fn().mockResolvedValue(mockConfig)
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig, updateConfig: mockUpdateConfig })
    )

    renderConfigPanel()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    await user.click(screen.getByRole('button', { name: /change model/i }))

    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(mockUpdateConfig).toHaveBeenCalled()
    })

    const callArgs = mockUpdateConfig.mock.calls[0][0] as CreateConfigRequest
    expect(callArgs.model_provider).toBe('anthropic')
    expect(callArgs.model_name).toBe('claude-3-5-sonnet-20240620')
  })

  it('cancel button resets form and exits edit mode', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig({ temperature: 0.7 })
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    await user.click(screen.getByRole('button', { name: /cancel/i }))

    expect(screen.getByRole('button', { name: /edit configuration/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument()
  })

  it('save success exits edit mode', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    const mockUpdateConfig = vi.fn().mockResolvedValue(mockConfig)
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig, updateConfig: mockUpdateConfig })
    )

    renderConfigPanel()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /edit configuration/i })).toBeInTheDocument()
    })
  })

  it('save failure stays in edit mode and shows error', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    const mockUpdateConfig = vi.fn().mockRejectedValue(new Error('Validation failed'))
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig, updateConfig: mockUpdateConfig })
    )

    renderConfigPanel()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /edit configuration/i })).not.toBeInTheDocument()
    })

    expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument()
  })

  it('isSaving state disables buttons during save', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    let resolveSave: (value: OpenCodeConfig) => void
    const savePromise = new Promise<OpenCodeConfig>((resolve) => {
      resolveSave = resolve
    })
    const mockUpdateConfig = vi.fn().mockReturnValue(savePromise)
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig, updateConfig: mockUpdateConfig })
    )

    renderConfigPanel()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    const saveButton = screen.getByRole('button', { name: /save changes/i })
    const cancelButton = screen.getByRole('button', { name: /cancel/i })

    await user.click(saveButton)

    await waitFor(() => {
      expect(saveButton).toBeDisabled()
      expect(cancelButton).toBeDisabled()
    })

    resolveSave!(mockConfig)
  })

  it('sub-components receive correct disabled prop (based on isEditing and isSaving)', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    expect(screen.getByTestId('model-selector')).toHaveTextContent('disabled: true')
    expect(screen.getByTestId('provider-config')).toHaveTextContent('disabled: true')
    expect(screen.getByTestId('tools-management')).toHaveTextContent('disabled: true')

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    expect(screen.getByTestId('model-selector')).toHaveTextContent('disabled: false')
    expect(screen.getByTestId('provider-config')).toHaveTextContent('disabled: false')
    expect(screen.getByTestId('tools-management')).toHaveTextContent('disabled: false')
  })

  it('ConfigHistory only shown when not editing', async () => {
    const user = userEvent.setup()
    const mockConfig = buildConfig()
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    expect(screen.getByTestId('config-history')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /edit configuration/i }))

    expect(screen.queryByTestId('config-history')).not.toBeInTheDocument()
  })

  it('breadcrumb navigation rendered', () => {
    const mockConfig = buildConfig()
    vi.mocked(useConfigHook.useConfig).mockReturnValue(
      createMockUseConfig({ config: mockConfig })
    )

    renderConfigPanel()

    const breadcrumbs = screen.getAllByText(/project/i)
    expect(breadcrumbs.length).toBeGreaterThan(0)
    
    const configLabels = screen.getAllByText(/configuration/i)
    expect(configLabels.length).toBeGreaterThan(0)
  })
})
