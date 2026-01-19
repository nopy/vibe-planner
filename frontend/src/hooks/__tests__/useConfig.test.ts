import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'

import { useConfig } from '../useConfig'
import * as api from '@/services/api'
import { buildConfig, buildConfigHistory } from '@/tests/factories/opencodeConfig'
import type { CreateConfigRequest } from '@/types'

vi.mock('@/services/api', () => ({
  getActiveConfig: vi.fn(),
  createOrUpdateConfig: vi.fn(),
  rollbackConfig: vi.fn(),
  getConfigHistory: vi.fn(),
}))

describe('useConfig', () => {
  const projectId = 'test-project-123'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('auto-fetches config on mount - success', async () => {
    const mockConfig = buildConfig()
    vi.mocked(api.getActiveConfig).mockResolvedValue(mockConfig)

    const { result } = renderHook(() => useConfig(projectId))

    expect(result.current.loading).toBe(true)
    expect(result.current.config).toBeNull()

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(api.getActiveConfig).toHaveBeenCalledWith(projectId)
    expect(result.current.config).toEqual(mockConfig)
    expect(result.current.error).toBeNull()
  })

  it('auto-fetch on mount - error (non-404)', async () => {
    const errorMessage = 'Internal server error'
    vi.mocked(api.getActiveConfig).mockRejectedValue({
      response: {
        status: 500,
        data: { error: errorMessage },
      },
    })

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.config).toBeNull()
    expect(result.current.error).toBe(errorMessage)
  })

  it('auto-fetch on mount - 404 (sets config to null, no error)', async () => {
    vi.mocked(api.getActiveConfig).mockRejectedValue({
      response: {
        status: 404,
      },
    })

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.config).toBeNull()
    expect(result.current.error).toBeNull()
  })

  it('refetch() success updates config', async () => {
    const initialConfig = buildConfig({ version: 1 })
    const updatedConfig = buildConfig({ version: 2 })

    vi.mocked(api.getActiveConfig)
      .mockResolvedValueOnce(initialConfig)
      .mockResolvedValueOnce(updatedConfig)

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
    })

    await result.current.refetch()

    await waitFor(() => {
      expect(result.current.config).toEqual(updatedConfig)
    })

    expect(api.getActiveConfig).toHaveBeenCalledTimes(2)
  })

  it('updateConfig() success updates state', async () => {
    const initialConfig = buildConfig()
    const updatedConfig = buildConfig({ temperature: 0.9 })
    const updateData: CreateConfigRequest = {
      model_provider: 'openai',
      model_name: 'gpt-4o-mini',
      temperature: 0.9,
      max_tokens: 4000,
      enabled_tools: ['file_ops'],
      max_iterations: 10,
      timeout_seconds: 300,
      api_key: 'sk-test',
    }

    vi.mocked(api.getActiveConfig).mockResolvedValue(initialConfig)
    vi.mocked(api.createOrUpdateConfig).mockResolvedValue(updatedConfig)

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
    })

    let returnedConfig: OpenCodeConfig | undefined
    await waitFor(async () => {
      returnedConfig = await result.current.updateConfig(updateData)
    })

    await waitFor(() => {
      expect(result.current.config).toEqual(updatedConfig)
      expect(result.current.error).toBeNull()
    })

    expect(returnedConfig).toEqual(updatedConfig)
    expect(api.createOrUpdateConfig).toHaveBeenCalledWith(projectId, updateData)
  })

  it('updateConfig() error surfaces error and preserves config', async () => {
    const initialConfig = buildConfig()
    const errorMessage = 'Validation failed'
    const updateData: CreateConfigRequest = {
      model_provider: 'openai',
      model_name: 'gpt-4o-mini',
      temperature: 0.7,
      max_tokens: 4000,
      enabled_tools: [],
      max_iterations: 10,
      timeout_seconds: 300,
      api_key: 'sk-test',
    }

    vi.mocked(api.getActiveConfig).mockResolvedValue(initialConfig)
    vi.mocked(api.createOrUpdateConfig).mockRejectedValue({
      response: {
        data: { error: errorMessage },
      },
    })

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
    })

    await waitFor(async () => {
      await expect(result.current.updateConfig(updateData)).rejects.toThrow()
    })

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
      expect(result.current.error).toBe(errorMessage)
    })
  })

  it('rollbackConfig() success triggers refetch', async () => {
    const initialConfig = buildConfig({ version: 3 })
    const rolledBackConfig = buildConfig({ version: 2 })

    vi.mocked(api.getActiveConfig)
      .mockResolvedValueOnce(initialConfig)
      .mockResolvedValueOnce(rolledBackConfig)
    vi.mocked(api.rollbackConfig).mockResolvedValue(undefined)

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
    })

    await result.current.rollbackConfig(2)

    await waitFor(() => {
      expect(result.current.config).toEqual(rolledBackConfig)
    })

    expect(api.rollbackConfig).toHaveBeenCalledWith(projectId, 2)
    expect(api.getActiveConfig).toHaveBeenCalledTimes(2)
    expect(result.current.error).toBeNull()
  })

  it('rollbackConfig() error surfaces error', async () => {
    const initialConfig = buildConfig()
    const errorMessage = 'Rollback failed'

    vi.mocked(api.getActiveConfig).mockResolvedValue(initialConfig)
    vi.mocked(api.rollbackConfig).mockRejectedValue({
      response: {
        data: { error: errorMessage },
      },
    })

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.config).toEqual(initialConfig)
    })

    await waitFor(async () => {
      await expect(result.current.rollbackConfig(1)).rejects.toThrow()
    })

    await waitFor(() => {
      expect(result.current.error).toBe(errorMessage)
    })
  })

  it('changing projectId triggers new fetch', async () => {
    const config1 = buildConfig({ project_id: 'project-1' })
    const config2 = buildConfig({ project_id: 'project-2' })

    vi.mocked(api.getActiveConfig)
      .mockResolvedValueOnce(config1)
      .mockResolvedValueOnce(config2)

    const { result, rerender } = renderHook(({ id }) => useConfig(id), {
      initialProps: { id: 'project-1' },
    })

    await waitFor(() => {
      expect(result.current.config).toEqual(config1)
    })

    rerender({ id: 'project-2' })

    await waitFor(() => {
      expect(result.current.config).toEqual(config2)
    })

    expect(api.getActiveConfig).toHaveBeenCalledWith('project-1')
    expect(api.getActiveConfig).toHaveBeenCalledWith('project-2')
  })

  it('error state clears on subsequent success', async () => {
    const mockConfig = buildConfig()

    vi.mocked(api.getActiveConfig)
      .mockRejectedValueOnce({
        response: {
          status: 500,
          data: { error: 'Server error' },
        },
      })
      .mockResolvedValueOnce(mockConfig)

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.error).toBe('Server error')
    })

    await result.current.refetch()

    await waitFor(() => {
      expect(result.current.config).toEqual(mockConfig)
      expect(result.current.error).toBeNull()
    })
  })

  it('loading states transition correctly', async () => {
    const mockConfig = buildConfig()
    vi.mocked(api.getActiveConfig).mockResolvedValue(mockConfig)

    const { result } = renderHook(() => useConfig(projectId))

    expect(result.current.loading).toBe(true)

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.config).toEqual(mockConfig)
  })

  it('hook provides all return values', async () => {
    const mockConfig = buildConfig()
    vi.mocked(api.getActiveConfig).mockResolvedValue(mockConfig)

    const { result } = renderHook(() => useConfig(projectId))

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current).toHaveProperty('config')
    expect(result.current).toHaveProperty('loading')
    expect(result.current).toHaveProperty('error')
    expect(result.current).toHaveProperty('updateConfig')
    expect(result.current).toHaveProperty('rollbackConfig')
    expect(result.current).toHaveProperty('refetch')
    expect(typeof result.current.updateConfig).toBe('function')
    expect(typeof result.current.rollbackConfig).toBe('function')
    expect(typeof result.current.refetch).toBe('function')
  })
})
