import { useDraggable } from '@dnd-kit/core'

import type { Task } from '@/types'

interface TaskCardProps {
  task: Task
  onClick: () => void
}

export function TaskCard({ task, onClick }: TaskCardProps) {
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

      <div className="flex items-center justify-between mt-2">
        {getPriorityBadge()}
        <span className="text-xs text-gray-400 font-mono">#{task.position}</span>
      </div>
    </div>
  )
}
