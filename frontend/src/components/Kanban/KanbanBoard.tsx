import { useState, useEffect } from 'react'
import {
  DndContext,
  DragEndEvent,
  DragOverlay,
  DragStartEvent,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'

import { moveTask, executeTask } from '@/services/api'
import { useTaskUpdates } from '@/hooks/useTaskUpdates'
import { KanbanColumn } from '@/components/Kanban/KanbanColumn'
import { TaskCard } from '@/components/Kanban/TaskCard'
import { CreateTaskModal } from '@/components/Kanban/CreateTaskModal'
import { TaskDetailPanel } from '@/components/Kanban/TaskDetailPanel'
import type { Task, TaskStatus, TaskExecutionState } from '@/types'

interface KanbanBoardProps {
  projectId: string
}

const COLUMNS: { id: TaskStatus; title: string }[] = [
  { id: 'todo', title: 'To Do' },
  { id: 'in_progress', title: 'In Progress' },
  { id: 'ai_review', title: 'AI Review' },
  { id: 'human_review', title: 'Human Review' },
  { id: 'done', title: 'Done' },
]

export function KanbanBoard({ projectId }: KanbanBoardProps) {
  const [localTasks, setLocalTasks] = useState<Task[]>([])
  const [activeTask, setActiveTask] = useState<Task | null>(null)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null)
  const [moveError, setMoveError] = useState<string | null>(null)
  const [executionStates, setExecutionStates] = useState<Record<string, TaskExecutionState>>({})

  const { tasks: wsTasks, isConnected, error: wsError, reconnect } = useTaskUpdates(projectId)

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(TouchSensor),
    useSensor(KeyboardSensor)
  )

  useEffect(() => {
    if (wsTasks) {
      setLocalTasks(wsTasks)
    }
  }, [wsTasks])

  const tasks = localTasks.length > 0 ? localTasks : wsTasks || []
  const isLoading = wsTasks === null && !wsError
  const error = wsError || moveError

  const handleDragStart = (event: DragStartEvent) => {
    const task = tasks.find(t => t.id === event.active.id)
    if (task) setActiveTask(task)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event
    setActiveTask(null)

    if (!over) return

    const taskId = active.id as string
    const newStatus = over.id as TaskStatus
    const task = tasks.find(t => t.id === taskId)

    if (!task || task.status === newStatus) return

    const previousTasks = [...tasks]

    setLocalTasks(currentTasks =>
      currentTasks.map(t => (t.id === taskId ? { ...t, status: newStatus } : t))
    )

    try {
      setMoveError(null)
      await moveTask(projectId, taskId, {
        status: newStatus,
        position: 0,
      })
    } catch (err) {
      console.error('Failed to move task:', err)
      setLocalTasks(previousTasks)
      setMoveError('Failed to update task status. Changes reverted.')
      setTimeout(() => setMoveError(null), 5000)
    }
  }

  const handleAddTask = () => {
    setIsCreateModalOpen(true)
  }

  const handleTaskCreated = () => {
    setIsCreateModalOpen(false)
  }

  const handleTaskClick = (taskId: string) => {
    setSelectedTaskId(taskId)
  }

  const handleTaskUpdated = (updatedTask: Task) => {
    setLocalTasks(currentTasks =>
      currentTasks.map(t => (t.id === updatedTask.id ? updatedTask : t))
    )
  }

  const handleTaskDeleted = (deletedTaskId: string) => {
    setLocalTasks(currentTasks => currentTasks.filter(t => t.id !== deletedTaskId))
    setSelectedTaskId(null)
  }

  const handleExecuteTask = async (taskId: string) => {
    setExecutionStates(prev => ({
      ...prev,
      [taskId]: { isExecuting: true, sessionId: null, error: null },
    }))

    try {
      const result = await executeTask(projectId, taskId)

      setExecutionStates(prev => ({
        ...prev,
        [taskId]: {
          isExecuting: true,
          sessionId: result.session_id,
          error: null,
        },
      }))
    } catch (err) {
      console.error('Failed to execute task:', err)
      setMoveError('Failed to execute task. Please try again.')

      setExecutionStates(prev => ({
        ...prev,
        [taskId]: {
          isExecuting: false,
          sessionId: null,
          error: 'Execution failed',
        },
      }))

      setTimeout(() => setMoveError(null), 5000)
    }
  }

  useEffect(() => {
    const currentTasks = localTasks.length > 0 ? localTasks : wsTasks || []
    setExecutionStates(prev => {
      const newStates = { ...prev }
      currentTasks.forEach(task => {
        if (
          newStates[task.id]?.isExecuting &&
          (task.status === 'done' || task.status === 'ai_review')
        ) {
          delete newStates[task.id]
        }
      })
      return newStates
    })
  }, [localTasks, wsTasks])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <svg
          className="animate-spin h-8 w-8 text-blue-600"
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
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div className="ml-3 flex-1">
            <h3 className="text-sm font-medium text-red-800">
              {wsError ? 'WebSocket connection error' : 'Error updating task'}
            </h3>
            <div className="mt-2 text-sm text-red-700">
              <p>{error}</p>
            </div>
            {wsError && (
              <div className="mt-4">
                <button
                  type="button"
                  onClick={reconnect}
                  className="bg-red-100 text-red-800 px-3 py-2 rounded-md text-sm font-medium hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
                >
                  Reconnect
                </button>
              </div>
            )}
          </div>
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <div
              className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}
            />
            {isConnected ? 'Live' : 'Offline'}
          </div>
        </div>
      </div>
    )
  }

  return (
    <>
      {moveError && !wsError && (
        <div className="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg p-3 flex items-center justify-between">
          <div className="flex items-center gap-2 text-sm text-yellow-800">
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                clipRule="evenodd"
              />
            </svg>
            {moveError}
          </div>
          <button
            onClick={() => setMoveError(null)}
            className="text-yellow-600 hover:text-yellow-800"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        </div>
      )}

      {!isConnected && !wsError && (
        <div className="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg p-3 flex items-center gap-2 text-sm text-yellow-800">
          <div className="w-2 h-2 rounded-full bg-yellow-500 animate-pulse" />
          Reconnecting to live updates...
        </div>
      )}

      <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4 h-full min-h-[calc(100vh-200px)]">
          {COLUMNS.map(column => (
            <KanbanColumn
              key={column.id}
              title={column.title}
              status={column.id}
              tasks={tasks.filter(t => t.status === column.id)}
              onAddTask={handleAddTask}
              onTaskClick={handleTaskClick}
              onExecute={handleExecuteTask}
              executionStates={executionStates}
            />
          ))}
        </div>

        <DragOverlay>
          {activeTask ? <TaskCard task={activeTask} onClick={() => {}} /> : null}
        </DragOverlay>
      </DndContext>

      <CreateTaskModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onTaskCreated={handleTaskCreated}
        projectId={projectId}
      />

      <TaskDetailPanel
        isOpen={!!selectedTaskId}
        taskId={selectedTaskId}
        projectId={projectId}
        onClose={() => setSelectedTaskId(null)}
        onTaskUpdated={handleTaskUpdated}
        onTaskDeleted={handleTaskDeleted}
        onExecute={handleExecuteTask}
        isExecuting={selectedTaskId ? executionStates[selectedTaskId]?.isExecuting || false : false}
        sessionId={selectedTaskId ? executionStates[selectedTaskId]?.sessionId || null : null}
      />
    </>
  )
}
