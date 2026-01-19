import { useState, useEffect } from 'react'

import { useParams, Link } from 'react-router-dom'

import { ConfigHistory } from '@/components/Config/ConfigHistory'
import { ModelSelector } from '@/components/Config/ModelSelector'
import { ProviderConfig } from '@/components/Config/ProviderConfig'
import { ToolsManagement } from '@/components/Config/ToolsManagement'
import { useConfig } from '@/hooks/useConfig'
import type { CreateConfigRequest } from '@/types'

export function ConfigPanel() {
  const { id: projectId } = useParams<{ id: string }>()
  const { config, loading, error, updateConfig, rollbackConfig } = useConfig(projectId || '')
  
  const [isEditing, setIsEditing] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [formData, setFormData] = useState<CreateConfigRequest>({
    model_provider: 'openai',
    model_name: 'gpt-4o-mini',
    temperature: 0.7,
    max_tokens: 4000,
    enabled_tools: [],
    max_iterations: 10,
    timeout_seconds: 300,
    api_key: '',
    api_endpoint: '',
    system_prompt: '',
    tools_config: {}
  })

  useEffect(() => {
    if (config) {
      setFormData({
        model_provider: config.model_provider,
        model_name: config.model_name,
        model_version: config.model_version,
        temperature: config.temperature,
        max_tokens: config.max_tokens,
        enabled_tools: config.enabled_tools,
        max_iterations: config.max_iterations,
        timeout_seconds: config.timeout_seconds,
        api_key: '',
        api_endpoint: config.api_endpoint,
        system_prompt: config.system_prompt,
        tools_config: config.tools_config
      })
    }
  }, [config])

  const handleSave = async () => {
    if (!projectId) return

    setIsSaving(true)
    try {
      await updateConfig(formData)
      setIsEditing(false)
    } catch (err) {
      console.error('Failed to save config:', err)
    } finally {
      setIsSaving(false)
    }
  }

  const handleCancel = () => {
    if (config) {
      setFormData({
        model_provider: config.model_provider,
        model_name: config.model_name,
        model_version: config.model_version,
        temperature: config.temperature,
        max_tokens: config.max_tokens,
        enabled_tools: config.enabled_tools,
        max_iterations: config.max_iterations,
        timeout_seconds: config.timeout_seconds,
        api_key: '',
        api_endpoint: config.api_endpoint,
        system_prompt: config.system_prompt,
        tools_config: config.tools_config
      })
    }
    setIsEditing(false)
  }

  if (loading && !config) {
    return (
      <div className="flex justify-center items-center h-64">
        <svg className="animate-spin h-8 w-8 text-blue-600" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>
    )
  }

  if (error && !config) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <p className="text-red-600 font-medium mb-2">Failed to load configuration</p>
        <p className="text-red-500 text-sm">{error}</p>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto pb-12">
      <div className="flex items-center justify-between mb-8">
        <div>
          <nav className="text-sm text-gray-500 mb-2">
            <Link to={`/projects/${projectId}`} className="hover:text-blue-600">Project</Link>
            <span className="mx-2">/</span>
            <span className="text-gray-900 font-medium">Configuration</span>
          </nav>
          <h1 className="text-2xl font-bold text-gray-900">OpenCode Configuration</h1>
          <p className="text-gray-500 mt-1">
            Configure AI models, tools, and execution parameters for this project.
          </p>
        </div>
        
        <div className="flex gap-3">
          {isEditing ? (
            <>
              <button
                onClick={handleCancel}
                disabled={isSaving}
                className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={isSaving}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
              >
                {isSaving && (
                  <svg className="animate-spin h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                )}
                Save Changes
              </button>
            </>
          ) : (
            <button
              onClick={() => setIsEditing(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
              Edit Configuration
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="mb-6 bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-600">
          {error}
        </div>
      )}

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="p-6 space-y-8">
          <ModelSelector
            value={{
              provider: isEditing ? formData.model_provider : (config?.model_provider || 'openai'),
              name: isEditing ? formData.model_name : (config?.model_name || 'gpt-4o-mini')
            }}
            onChange={(provider, name) => setFormData(prev => ({
              ...prev,
              model_provider: provider,
              model_name: name
            }))}
            disabled={!isEditing || isSaving}
          />

          <ProviderConfig
            provider={isEditing ? formData.model_provider : (config?.model_provider || 'openai')}
            apiKey={formData.api_key || ''}
            apiEndpoint={formData.api_endpoint}
            temperature={isEditing ? formData.temperature : (config?.temperature || 0.7)}
            maxTokens={isEditing ? formData.max_tokens : (config?.max_tokens || 4000)}
            onChange={(field, value) => setFormData(prev => ({
              ...prev,
              [field]: value
            }))}
            disabled={!isEditing || isSaving}
          />

          <ToolsManagement
            enabledTools={isEditing ? formData.enabled_tools : (config?.enabled_tools || [])}
            toolsConfig={isEditing ? formData.tools_config : (config?.tools_config)}
            onChange={(enabledTools, toolsConfig) => setFormData(prev => ({
              ...prev,
              enabled_tools: enabledTools,
              ...(toolsConfig && { tools_config: toolsConfig })
            }))}
            disabled={!isEditing || isSaving}
          />
        </div>

        {!isEditing && projectId && (
          <div className="border-t border-gray-200 p-6 bg-gray-50">
            <ConfigHistory 
              projectId={projectId}
              onRollback={rollbackConfig}
            />
          </div>
        )}
      </div>
    </div>
  )
}
