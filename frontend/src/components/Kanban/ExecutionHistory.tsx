import { useState, useEffect } from 'react'

import { getTaskSessions } from '@/services/api'
import type { Session, SessionStatus } from '@/types'

interface ExecutionHistoryProps {
  projectId: string
  taskId: string
}

export function ExecutionHistory({ projectId, taskId }: ExecutionHistoryProps) {
  const [sessions, setSessions] = useState<Session[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [expandedSessionId, setExpandedSessionId] = useState<string | null>(null)

  useEffect(() => {
    const fetchSessions = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const data = await getTaskSessions(projectId, taskId)
        setSessions(data.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()))
      } catch (err) {
        console.error('Failed to fetch sessions:', err)
        setError('Failed to load execution history')
      } finally {
        setIsLoading(false)
      }
    }

    fetchSessions()
  }, [projectId, taskId])

  const getStatusBadge = (status: SessionStatus) => {
    const baseClasses = 'px-2 py-0.5 text-xs font-medium rounded-full uppercase tracking-wide'
    switch (status) {
      case 'completed':
        return <span className={`${baseClasses} bg-green-100 text-green-800`}>Completed</span>
      case 'running':
        return <span className={`${baseClasses} bg-blue-100 text-blue-800`}>Running</span>
      case 'failed':
        return <span className={`${baseClasses} bg-red-100 text-red-800`}>Failed</span>
      case 'cancelled':
        return <span className={`${baseClasses} bg-gray-100 text-gray-800`}>Cancelled</span>
      case 'pending':
        return <span className={`${baseClasses} bg-yellow-100 text-yellow-800`}>Pending</span>
      default:
        return null
    }
  }

  const formatDuration = (durationMs: number) => {
    if (durationMs === 0) return 'N/A'
    const seconds = Math.floor(durationMs / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`
    } else {
      return `${seconds}s`
    }
  }

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString()
  }

  const truncateOutput = (output: string | undefined, maxLength = 100) => {
    if (!output) return 'No output'
    if (output.length <= maxLength) return output
    return output.substring(0, maxLength) + '...'
  }

  const toggleSession = (sessionId: string) => {
    setExpandedSessionId(expandedSessionId === sessionId ? null : sessionId)
  }

  if (isLoading) {
    return (
      <div className="col-span-2 pt-4 border-t border-gray-200">
        <label className="block text-sm font-medium text-gray-500 mb-2">Execution History</label>
        <div className="flex items-center justify-center py-6">
          <svg
            className="animate-spin h-6 w-6 text-blue-600"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="col-span-2 pt-4 border-t border-gray-200">
        <label className="block text-sm font-medium text-gray-500 mb-2">Execution History</label>
        <div className="bg-red-50 border border-red-200 rounded-lg p-3">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      </div>
    )
  }

  if (sessions.length === 0) {
    return (
      <div className="col-span-2 pt-4 border-t border-gray-200">
        <label className="block text-sm font-medium text-gray-500 mb-2">Execution History</label>
        <div className="bg-gray-50 rounded-lg p-4 text-center">
          <p className="text-sm text-gray-500">No execution history yet</p>
        </div>
      </div>
    )
  }

  return (
    <div className="col-span-2 pt-4 border-t border-gray-200">
      <label className="block text-sm font-medium text-gray-500 mb-2">
        Execution History ({sessions.length})
      </label>
      <div className="space-y-2">
        {sessions.map(session => {
          const isExpanded = expandedSessionId === session.id
          return (
            <div key={session.id} className="border border-gray-200 rounded-lg overflow-hidden">
              <button
                onClick={() => toggleSession(session.id)}
                className="w-full px-3 py-2 bg-gray-50 hover:bg-gray-100 flex items-center justify-between gap-2 text-left transition-colors"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    {getStatusBadge(session.status)}
                    <span className="text-xs text-gray-500 font-mono">
                      {formatTimestamp(session.created_at)}
                    </span>
                  </div>
                  <p className="text-xs text-gray-600 truncate">
                    {truncateOutput(session.output || session.error)}
                  </p>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  {session.duration_ms > 0 && (
                    <span className="text-xs text-gray-500 font-medium">
                      {formatDuration(session.duration_ms)}
                    </span>
                  )}
                  <svg
                    className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 9l-7 7-7-7"
                    />
                  </svg>
                </div>
              </button>
              {isExpanded && (
                <div className="p-3 bg-white border-t border-gray-200">
                  <div className="space-y-2 text-sm">
                    <div>
                      <label className="block text-xs font-medium text-gray-500 mb-1">
                        Session ID
                      </label>
                      <p className="font-mono text-xs text-gray-700 bg-gray-50 px-2 py-1 rounded">
                        {session.id}
                      </p>
                    </div>
                    {session.started_at && (
                      <div>
                        <label className="block text-xs font-medium text-gray-500 mb-1">
                          Started At
                        </label>
                        <p className="text-xs text-gray-700">{formatTimestamp(session.started_at)}</p>
                      </div>
                    )}
                    {session.completed_at && (
                      <div>
                        <label className="block text-xs font-medium text-gray-500 mb-1">
                          Completed At
                        </label>
                        <p className="text-xs text-gray-700">
                          {formatTimestamp(session.completed_at)}
                        </p>
                      </div>
                    )}
                    {session.output && (
                      <div>
                        <label className="block text-xs font-medium text-gray-500 mb-1">
                          Output
                        </label>
                        <pre className="text-xs text-gray-700 bg-gray-50 px-2 py-1 rounded whitespace-pre-wrap break-words max-h-64 overflow-y-auto">
                          {session.output}
                        </pre>
                      </div>
                    )}
                    {session.error && (
                      <div>
                        <label className="block text-xs font-medium text-red-500 mb-1">Error</label>
                        <pre className="text-xs text-red-700 bg-red-50 px-2 py-1 rounded whitespace-pre-wrap break-words">
                          {session.error}
                        </pre>
                      </div>
                    )}
                    {session.prompt && (
                      <div>
                        <label className="block text-xs font-medium text-gray-500 mb-1">
                          Prompt
                        </label>
                        <pre className="text-xs text-gray-700 bg-gray-50 px-2 py-1 rounded whitespace-pre-wrap break-words">
                          {session.prompt}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
