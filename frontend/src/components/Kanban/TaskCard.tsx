import { useDraggable } from '@dnd-kit/core'

import type { Task } from '@/types'

interface TaskCardProps {
  task: Task
  onClick: () => void
  onExecute?: (taskId: string) => void
  isExecuting?: boolean
}

export function TaskCard({ task, onClick, onExecute, isExecuting = false }: TaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: task.id,
  })

  const style = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
      }
    : undefined

  const getPriorityBadge = () => {
    const baseClasses = 'px-2 py-0.5 text-xs font-medium rounded-full uppercase tracking-wide'
    switch (task.priority) {
      case 'high':
        return <span className={`${baseClasses} bg-red-100 text-red-800`}>High</span>
      case 'medium':
        return <span className={`${baseClasses} bg-yellow-100 text-yellow-800`}>Medium</span>
      case 'low':
        return <span className={`${baseClasses} bg-green-100 text-green-800`}>Low</span>
      default:
        return null
    }
  }

  const getExecutionBadge = () => {
    if (!isExecuting && !task.current_session_id) return null

    const baseClasses = 'px-2 py-0.5 text-xs font-medium rounded-full uppercase tracking-wide'
    if (isExecuting || task.status === 'in_progress') {
      return (
        <span className={`${baseClasses} bg-blue-100 text-blue-800 flex items-center gap-1`}>
          <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
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
          Running
        </span>
      )
    }
    return null
  }

  const handleExecute = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (onExecute && !isExecuting) {
      onExecute(task.id)
    }
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      onClick={onClick}
      className={`bg-white p-4 rounded-lg shadow-sm border border-gray-200 cursor-grab hover:shadow-md transition-shadow group mb-3 ${
        isDragging ? 'opacity-50 cursor-grabbing ring-2 ring-blue-500 ring-offset-2 rotate-2' : ''
      }`}
    >
      <div className="flex items-start justify-between gap-2 mb-2">
        <h4 className="text-sm font-medium text-gray-900 line-clamp-2 leading-tight">
          {task.title}
        </h4>
      </div>

      <div className="flex items-center justify-between gap-2 mt-2">
        <div className="flex items-center gap-2">
          {getPriorityBadge()}
          {getExecutionBadge()}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-400 font-mono">#{task.position}</span>
          {task.status === 'todo' && onExecute && (
            <button
              onClick={handleExecute}
              disabled={isExecuting}
              className="p-1 rounded hover:bg-blue-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              title={isExecuting ? 'Executing...' : 'Execute task'}
            >
              <svg
                className={`w-4 h-4 ${isExecuting ? 'text-blue-600' : 'text-gray-600 hover:text-blue-600'}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
