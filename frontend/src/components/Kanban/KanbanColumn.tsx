import { useDroppable } from '@dnd-kit/core'

import { TaskCard } from '@/components/Kanban/TaskCard'
import type { Task, TaskStatus } from '@/types'

interface KanbanColumnProps {
  title: string
  status: TaskStatus
  tasks: Task[]
  onAddTask: () => void
  onTaskClick: (taskId: string) => void
}

export function KanbanColumn({ title, status, tasks, onAddTask, onTaskClick }: KanbanColumnProps) {
  const { setNodeRef, isOver } = useDroppable({
    id: status,
  })

  return (
    <div className="flex flex-col h-full bg-gray-50/50 rounded-xl border border-gray-200/60 overflow-hidden">
      <div className="p-3 border-b border-gray-100 bg-white flex items-center justify-between sticky top-0 z-10">
        <div className="flex items-center gap-2">
          <h3 className="font-semibold text-gray-700 text-sm tracking-tight">{title}</h3>
          <span className="px-2 py-0.5 bg-gray-100 text-gray-600 text-xs font-medium rounded-full">
            {tasks.length}
          </span>
        </div>
        <button
          onClick={onAddTask}
          className="p-1 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded transition-colors"
          title="Add Task"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      <div
        ref={setNodeRef}
        className={`flex-1 p-3 overflow-y-auto min-h-[500px] transition-colors duration-200 ${
          isOver ? 'bg-blue-50/50 ring-2 ring-inset ring-blue-500/20' : ''
        }`}
      >
        <div className="flex flex-col gap-3">
          {tasks.map(task => (
            <TaskCard key={task.id} task={task} onClick={() => onTaskClick(task.id)} />
          ))}

          {tasks.length === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-gray-400 border-2 border-dashed border-gray-200 rounded-lg">
              <span className="text-sm">No tasks</span>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
