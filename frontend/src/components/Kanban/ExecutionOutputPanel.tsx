import { useEffect, useRef } from 'react'

import { useTaskExecution } from '@/hooks/useTaskExecution'

interface ExecutionOutputPanelProps {
  projectId: string
  taskId: string
  sessionId: string | null
  isExecuting: boolean
}

export function ExecutionOutputPanel({
  projectId,
  taskId,
  sessionId,
  isExecuting,
}: ExecutionOutputPanelProps) {
  const { output, isStreaming, error } = useTaskExecution(projectId, taskId, sessionId)
  const outputEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new output arrives
  useEffect(() => {
    if (outputEndRef.current) {
      outputEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [output])

  if (!sessionId && !isExecuting) {
    return null
  }

  return (
    <div className="mt-6 border border-gray-200 rounded-lg overflow-hidden">
      <div className="bg-gray-800 px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex gap-1.5">
            <div className="w-3 h-3 rounded-full bg-red-500"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
          </div>
          <span className="text-gray-300 text-sm font-mono ml-3">Execution Output</span>
        </div>

        <div className="flex items-center gap-3">
          {isStreaming && (
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse"></div>
              <span className="text-xs text-green-400 font-mono">LIVE</span>
            </div>
          )}
          {error && (
            <div className="flex items-center gap-2">
              <div className="w-2 h-2 rounded-full bg-red-400"></div>
              <span className="text-xs text-red-400 font-mono">CONNECTION ERROR</span>
            </div>
          )}
        </div>
      </div>

      <div className="bg-gray-900 p-4 h-96 overflow-y-auto font-mono text-sm">
        {output.length === 0 && isExecuting && (
          <div className="text-gray-500 flex items-center gap-2">
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
                fill="none"
              ></circle>
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <span>Waiting for output...</span>
          </div>
        )}

        {output.map((event, index) => (
          <div key={index} className={`mb-1 ${getEventClass(event.type)}`}>
            <span className="text-gray-500 select-none">
              [{new Date(event.timestamp).toLocaleTimeString()}]{' '}
            </span>
            <span className="whitespace-pre-wrap break-words">{event.data}</span>
          </div>
        ))}

        <div ref={outputEndRef} />
      </div>
    </div>
  )
}

function getEventClass(type: string): string {
  switch (type) {
    case 'output':
      return 'text-gray-100'
    case 'error':
      return 'text-red-400'
    case 'status':
      return 'text-blue-400'
    case 'done':
      return 'text-green-400 font-semibold'
    default:
      return 'text-gray-300'
  }
}
