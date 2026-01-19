import { useState, useEffect } from 'react'

import { getConfigHistory } from '@/services/api'
import type { OpenCodeConfig } from '@/types'

interface ConfigHistoryProps {
  projectId: string
  onRollback: (version: number) => Promise<void>
}

export function ConfigHistory({ projectId, onRollback }: ConfigHistoryProps) {
  const [history, setHistory] = useState<OpenCodeConfig[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedVersion, setExpandedVersion] = useState<number | null>(null)
  const [isRollingBack, setIsRollingBack] = useState(false)

  useEffect(() => {
    const fetchHistory = async () => {
      setIsLoading(true)
      try {
        const data = await getConfigHistory(projectId)
        setHistory(data)
      } catch (err) {
        console.error('Failed to fetch config history:', err)
        setError('Failed to load version history')
      } finally {
        setIsLoading(false)
      }
    }

    if (projectId) {
      fetchHistory()
    }
  }, [projectId])

  const handleRollback = async (version: number) => {
    if (!window.confirm(`Are you sure you want to rollback to version ${version}? Current configuration will be overwritten.`)) {
      return
    }

    setIsRollingBack(true)
    try {
      await onRollback(version)
    } catch (err) {
      console.error('Failed to rollback:', err)
    } finally {
      setIsRollingBack(false)
    }
  }

  const toggleDetails = (version: number) => {
    setExpandedVersion(expandedVersion === version ? null : version)
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <svg className="animate-spin h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-600">
        {error}
      </div>
    )
  }

  if (history.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No configuration history found.
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-gray-800 mb-4">Version History</h3>
      
      <div className="space-y-3">
        {history.map((config) => (
          <div 
            key={config.version}
            className={`border rounded-lg overflow-hidden transition-colors ${
              config.is_active 
                ? 'border-blue-200 bg-blue-50' 
                : 'border-gray-200 bg-white hover:bg-gray-50'
            }`}
          >
            <div className="flex items-center justify-between p-4">
              <div className="flex items-center gap-4">
                <div className={`flex items-center justify-center w-8 h-8 rounded-full font-bold text-sm ${
                  config.is_active ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-600'
                }`}>
                  v{config.version}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900">
                      {new Date(config.created_at).toLocaleString()}
                    </span>
                    {config.is_active && (
                      <span className="px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 rounded-full">
                        Active
                      </span>
                    )}
                  </div>
                  <div className="text-sm text-gray-500 mt-0.5">
                    {config.model_provider} / {config.model_name}
                  </div>
                </div>
              </div>
              
              <div className="flex items-center gap-2">
                {!config.is_active && (
                  <button
                    onClick={() => handleRollback(config.version)}
                    disabled={isRollingBack}
                    className="px-3 py-1 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded transition-colors disabled:opacity-50"
                  >
                    Rollback
                  </button>
                )}
                <button
                  onClick={() => toggleDetails(config.version)}
                  className="p-1 text-gray-400 hover:text-gray-600 rounded-full hover:bg-gray-100"
                >
                  <svg 
                    className={`w-5 h-5 transform transition-transform ${expandedVersion === config.version ? 'rotate-180' : ''}`} 
                    fill="none" 
                    stroke="currentColor" 
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
              </div>
            </div>

            {expandedVersion === config.version && (
              <div className="px-4 pb-4 border-t border-gray-100 pt-3 bg-gray-50">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="block text-gray-500 text-xs uppercase tracking-wide">Temperature</span>
                    <span className="font-medium">{config.temperature}</span>
                  </div>
                  <div>
                    <span className="block text-gray-500 text-xs uppercase tracking-wide">Max Tokens</span>
                    <span className="font-medium">{config.max_tokens}</span>
                  </div>
                  <div className="col-span-2">
                    <span className="block text-gray-500 text-xs uppercase tracking-wide mb-1">Enabled Tools</span>
                    <div className="flex flex-wrap gap-2">
                      {config.enabled_tools && config.enabled_tools.length > 0 ? (
                        config.enabled_tools.map(tool => (
                          <span key={tool} className="px-2 py-1 bg-white border border-gray-200 rounded text-xs text-gray-700">
                            {tool}
                          </span>
                        ))
                      ) : (
                        <span className="text-gray-400 italic">No tools enabled</span>
                      )}
                    </div>
                  </div>
                  <div className="col-span-2">
                    <span className="block text-gray-500 text-xs uppercase tracking-wide">Updated By</span>
                    <span className="font-medium">{config.created_by}</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
