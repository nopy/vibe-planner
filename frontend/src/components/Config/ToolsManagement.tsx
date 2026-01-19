import { useState, useEffect } from 'react'

interface ToolsManagementProps {
  enabledTools: string[]
  toolsConfig?: Record<string, unknown>
  onChange: (enabledTools: string[], toolsConfig?: Record<string, unknown>) => void
  disabled?: boolean
}

export function ToolsManagement({ enabledTools, onChange, disabled }: ToolsManagementProps) {
  const [selectedTools, setSelectedTools] = useState<string[]>(enabledTools)

  useEffect(() => {
    setSelectedTools(enabledTools)
  }, [enabledTools])

  const handleToggleTool = (toolId: string) => {
    if (disabled) return

    const newTools = selectedTools.includes(toolId)
      ? selectedTools.filter(t => t !== toolId)
      : [...selectedTools, toolId]
    
    setSelectedTools(newTools)
    onChange(newTools)
  }

  const tools = [
    {
      id: 'file_ops',
      name: 'File Operations',
      description: 'Read, write, and modify files in the workspace',
      icon: (
        <svg className="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      )
    },
    {
      id: 'web_search',
      name: 'Web Search',
      description: 'Search the web for information',
      icon: (
        <svg className="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
        </svg>
      )
    },
    {
      id: 'code_exec',
      name: 'Code Execution',
      description: 'Execute code snippets (Python, JavaScript, etc.)',
      icon: (
        <svg className="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
        </svg>
      )
    },
    {
      id: 'terminal',
      name: 'Terminal Access',
      description: 'Run shell commands in the workspace',
      icon: (
        <svg className="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
      )
    }
  ]

  return (
    <div className="space-y-6 pt-6 border-t border-gray-200">
      <h3 className="text-lg font-semibold text-gray-800 mb-2 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 text-blue-600" viewBox="0 0 20 20" fill="currentColor">
          <path fillRule="evenodd" d="M11.3 1.046A1 1 0 0112 2v5h4a1 1 0 01.82 1.573l-7 10A1 1 0 018 18v-5H4a1 1 0 01-.82-1.573l7-10a1 1 0 011.12-.38z" clipRule="evenodd" />
        </svg>
        Agent Capabilities
      </h3>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {tools.map((tool) => (
          <div
            key={tool.id}
            onClick={() => handleToggleTool(tool.id)}
            className={`
              relative flex items-start p-4 border rounded-lg cursor-pointer transition-colors
              ${selectedTools.includes(tool.id) 
                ? 'border-blue-500 bg-blue-50' 
                : 'border-gray-200 hover:bg-gray-50'
              }
              ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
            `}
          >
            <div className="flex items-center h-5">
              <input
                type="checkbox"
                checked={selectedTools.includes(tool.id)}
                onChange={() => handleToggleTool(tool.id)}
                disabled={disabled}
                className="focus:ring-blue-500 h-4 w-4 text-blue-600 border-gray-300 rounded cursor-pointer disabled:cursor-not-allowed"
              />
            </div>
            <div className="ml-3 flex-1">
              <div className="flex items-center gap-2">
                {tool.icon}
                <label className={`font-medium text-gray-700 cursor-pointer ${disabled ? 'cursor-not-allowed' : ''}`}>
                  {tool.name}
                </label>
              </div>
              <p className="text-sm text-gray-500 mt-1">{tool.description}</p>
            </div>
          </div>
        ))}
      </div>

      <p className="text-sm text-gray-500 italic">
        Tip: Disabling unused tools can reduce token usage and improve agent focus.
      </p>
    </div>
  )
}
