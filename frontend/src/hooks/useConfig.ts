import { useState, useEffect, useCallback } from 'react'

import {
  getActiveConfig,
  createOrUpdateConfig,
  rollbackConfig as rollbackConfigApi,
} from '@/services/api'
import type { OpenCodeConfig, CreateConfigRequest } from '@/types'

export interface UseConfigReturn {
  config: OpenCodeConfig | null
  loading: boolean
  error: string | null
  updateConfig: (data: CreateConfigRequest) => Promise<OpenCodeConfig>
  rollbackConfig: (version: number) => Promise<void>
  refetch: () => Promise<void>
}

export function useConfig(projectId: string): UseConfigReturn {
  const [config, setConfig] = useState<OpenCodeConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = useCallback(async () => {
    if (!projectId) {
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)
      const data = await getActiveConfig(projectId)
      setConfig(data)
    } catch (err: any) {
      if (err.response?.status === 404) {
        setConfig(null)
      } else {
        console.error('[useConfig] Failed to fetch config:', err)
        setError(err.response?.data?.error || 'Failed to load configuration')
      }
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => {
    fetchConfig()
  }, [fetchConfig])

  const updateConfig = useCallback(
    async (data: CreateConfigRequest): Promise<OpenCodeConfig> => {
      try {
        setLoading(true)
        setError(null)
        const newConfig = await createOrUpdateConfig(projectId, data)
        setConfig(newConfig)
        return newConfig
      } catch (err: any) {
        console.error('[useConfig] Failed to update config:', err)
        const errorMessage = err.response?.data?.error || 'Failed to update configuration'
        setError(errorMessage)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [projectId]
  )

  const rollbackConfig = useCallback(
    async (version: number): Promise<void> => {
      try {
        setLoading(true)
        setError(null)
        await rollbackConfigApi(projectId, version)
        await fetchConfig()
      } catch (err: any) {
        console.error('[useConfig] Failed to rollback config:', err)
        const errorMessage = err.response?.data?.error || 'Failed to rollback configuration'
        setError(errorMessage)
        throw err
      } finally {
        setLoading(false)
      }
    },
    [projectId, fetchConfig]
  )

  return {
    config,
    loading,
    error,
    updateConfig,
    rollbackConfig,
    refetch: fetchConfig,
  }
}
