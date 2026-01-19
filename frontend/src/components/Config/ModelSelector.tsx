import { useState, useEffect } from 'react'

interface ModelSelectorProps {
  value: {
    provider: string
    name: string
  }
  onChange: (provider: string, name: string) => void
  disabled?: boolean
}

export function ModelSelector({ value, onChange, disabled }: ModelSelectorProps) {
  const [provider, setProvider] = useState(value.provider)
  const [modelName, setModelName] = useState(value.name)

  useEffect(() => {
    setProvider(value.provider)
    setModelName(value.name)
  }, [value.provider, value.name])

  const handleProviderChange = (newProvider: string) => {
    setProvider(newProvider)
    
    let defaultModel = ''
    if (newProvider === 'openai') {
      defaultModel = 'gpt-4o-mini'
    } else if (newProvider === 'anthropic') {
      defaultModel = 'claude-3-5-sonnet-20240620'
    }
    
    setModelName(defaultModel)
    onChange(newProvider, defaultModel)
  }

  const handleModelChange = (newModel: string) => {
    setModelName(newModel)
    onChange(provider, newModel)
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-gray-800 mb-2 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-blue-600" viewBox="0 0 20 20" fill="currentColor">
          <path fillRule="evenodd" d="M10 2a1 1 0 011 1v1.323l3.954 1.582 1.699-3.181a1 1 0 111.768.951l-1.778 3.328 3.328 1.778a1 1 0 11-.951 1.767l-3.181-1.699L12.323 11H13a1 1 0 110 2h-1.323l-1.582 3.954 3.181 1.699a1 1 0 01-.951 1.768l-3.328-1.778-1.778 3.328a1 1 0 01-1.768-.951l1.699-3.181L3.677 14H3a1 1 0 110-2h1.323l1.582-3.954-3.181-1.699a1 1 0 01.951-1.768l3.328 1.778 1.778-3.328A1 1 0 019 3.677V3a1 1 0 011-1zm-1 5a1 1 0 011-1h.01a1 1 0 110 2H10a1 1 0 01-1-1z" clipRule="evenodd" />
        </svg>
        AI Model Selection
      </h3>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Provider
          </label>
          <div className="relative">
            <select
              value={provider}
              onChange={(e) => handleProviderChange(e.target.value)}
              disabled={disabled}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 appearance-none bg-white"
            >
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="custom">Custom / Self-Hosted</option>
            </select>
            <div className="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none">
              <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Model
          </label>
          {provider === 'custom' ? (
            <input
              type="text"
              value={modelName}
              onChange={(e) => handleModelChange(e.target.value)}
              disabled={disabled}
              placeholder="e.g. llama-3-70b"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
            />
          ) : (
            <div className="relative">
              <select
                value={modelName}
                onChange={(e) => handleModelChange(e.target.value)}
                disabled={disabled}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 appearance-none bg-white"
              >
                {provider === 'openai' && (
                  <>
                    <option value="gpt-4o-mini">GPT-4o Mini (128k context, $0.15/$0.60 per 1M tokens) - Recommended</option>
                    <option value="gpt-4o">GPT-4o (128k context, $5.00/$15.00 per 1M tokens)</option>
                    <option value="gpt-4">GPT-4 (8k context)</option>
                    <option value="gpt-3.5-turbo">GPT-3.5 Turbo (16k context)</option>
                  </>
                )}
                {provider === 'anthropic' && (
                  <>
                    <option value="claude-3-5-sonnet-20240620">Claude 3.5 Sonnet (200k context, $3/$15) - Recommended</option>
                    <option value="claude-3-opus-20240229">Claude 3 Opus (200k context, $15/$75)</option>
                    <option value="claude-3-sonnet-20240229">Claude 3 Sonnet (200k context, $3/$15)</option>
                    <option value="claude-3-haiku-20240307">Claude 3 Haiku (200k context, $0.25/$1.25)</option>
                  </>
                )}
              </select>
              <div className="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none">
                <svg className="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
                </svg>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
