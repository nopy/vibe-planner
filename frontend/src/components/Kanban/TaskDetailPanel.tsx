import { useState, useEffect } from 'react'

import { ExecutionOutputPanel } from '@/components/Kanban/ExecutionOutputPanel'
import { getTask, updateTask, deleteTask } from '@/services/api'
import type { Task, UpdateTaskRequest, TaskPriority } from '@/types'

interface TaskDetailPanelProps {
  isOpen: boolean
  taskId: string | null
  projectId: string
  onClose: () => void
  onTaskUpdated: (task: Task) => void
  onTaskDeleted: (taskId: string) => void
  onExecute?: (taskId: string) => void
  isExecuting?: boolean
  sessionId?: string | null
}

export function TaskDetailPanel({
  isOpen,
  taskId,
  projectId,
  onClose,
  onTaskUpdated,
  onTaskDeleted,
  onExecute,
  isExecuting = false,
  sessionId = null,
}: TaskDetailPanelProps) {
  const [task, setTask] = useState<Task | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [isEditing, setIsEditing] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [editFormData, setEditFormData] = useState<{
    title: string
    description: string
    priority: TaskPriority
  }>({
    title: '',
    description: '',
    priority: 'medium',
  })

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        handleClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  useEffect(() => {
    if (!isOpen || !taskId) {
      if (!isOpen) {
        const timer = setTimeout(() => {
          setTask(null)
          setIsEditing(false)
          setShowDeleteConfirm(false)
          setError(null)
        }, 300)
        return () => clearTimeout(timer)
      }
      return
    }

    const fetchTask = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const data = await getTask(projectId, taskId)
        setTask(data)
        setEditFormData({
          title: data.title,
          description: data.description || '',
          priority: data.priority,
        })
      } catch (err) {
        console.error('Failed to fetch task:', err)
        setError('Failed to load task. Please try again.')
      } finally {
        setIsLoading(false)
      }
    }

    fetchTask()
  }, [isOpen, taskId, projectId])

  const handleClose = () => {
    onClose()
    setTimeout(() => {
      setIsEditing(false)
      setShowDeleteConfirm(false)
      setError(null)
    }, 300)
  }

  const handleStartEdit = () => {
    if (!task) return
    setEditFormData({
      title: task.title,
      description: task.description || '',
      priority: task.priority,
    })
    setIsEditing(true)
  }

  const handleCancelEdit = () => {
    if (!task) return
    setEditFormData({
      title: task.title,
      description: task.description || '',
      priority: task.priority,
    })
    setIsEditing(false)
  }

  const handleSave = async () => {
    if (!task || !taskId) return

    if (!editFormData.title.trim()) {
      return
    }

    setIsSaving(true)
    try {
      const payload: UpdateTaskRequest = {
        title: editFormData.title.trim(),
        description: editFormData.description.trim(),
        priority: editFormData.priority,
      }

      const updatedTask = await updateTask(projectId, taskId, payload)
      setTask(updatedTask)
      onTaskUpdated(updatedTask)
      setIsEditing(false)
    } catch (err) {
      console.error('Failed to update task:', err)
      setError('Failed to update task. Please try again.')
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!taskId) return

    setIsDeleting(true)
    try {
      await deleteTask(projectId, taskId)
      onTaskDeleted(taskId)
      onClose()
    } catch (err) {
      console.error('Failed to delete task:', err)
      setError('Failed to delete task. Please try again.')
      setShowDeleteConfirm(false)
    } finally {
      setIsDeleting(false)
    }
  }

  const handleExecute = () => {
    if (!taskId || !onExecute || isExecuting) return
    onExecute(taskId)
  }

  const getPriorityBadge = (priority: TaskPriority) => {
    const baseClasses = 'px-2 py-0.5 text-xs font-medium rounded-full uppercase tracking-wide'
    switch (priority) {
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

  if (!isOpen && !task) return null

  return (
    <>
      <div
        className={`fixed inset-0 bg-black bg-opacity-40 z-40 transition-opacity duration-300 ${
          isOpen ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        onClick={handleClose}
      />

      <aside
        className={`fixed right-0 top-0 h-full w-full max-w-md bg-white shadow-xl z-50 transform transition-transform duration-300 flex flex-col ${
          isOpen ? 'translate-x-0' : 'translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-xl font-bold text-gray-900">
            {isEditing ? 'Edit Task' : 'Task Details'}
          </h2>
          <div className="flex items-center gap-2">
            {!isEditing && task && !isLoading && task.status === 'todo' && onExecute && (
              <button
                onClick={handleExecute}
                disabled={isExecuting}
                className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              >
                <svg
                  className={`w-4 h-4 ${isExecuting ? 'animate-spin' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  {isExecuting ? (
                    <>
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
                    </>
                  ) : (
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M13 10V3L4 14h7v7l9-11h-7z"
                    />
                  )}
                </svg>
                {isExecuting ? 'Executing...' : 'Execute'}
              </button>
            )}
            {!isEditing && task && !isLoading && (
              <button
                onClick={handleStartEdit}
                className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 flex items-center gap-1"
              >
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                  />
                </svg>
                Edit
              </button>
            )}
            <button
              onClick={handleClose}
              className="text-gray-400 hover:text-gray-600 focus:outline-none"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>

        <div className="p-6 overflow-y-auto flex-1">
          {isLoading ? (
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
          ) : error ? (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
              <p className="text-sm text-red-600 mb-3">{error}</p>
              <button
                onClick={() => {
                  if (taskId) {
                    setIsLoading(true)
                    setError(null)
                    getTask(projectId, taskId)
                      .then(data => {
                        setTask(data)
                        setEditFormData({
                          title: data.title,
                          description: data.description || '',
                          priority: data.priority,
                        })
                        setIsLoading(false)
                      })
                      .catch(err => {
                        console.error('Failed to fetch task:', err)
                        setError('Failed to load task. Please try again.')
                        setIsLoading(false)
                      })
                  }
                }}
                className="text-sm text-red-700 font-medium hover:text-red-800 underline"
              >
                Try Again
              </button>
            </div>
          ) : !task ? (
            <div className="text-center py-12">
              <p className="text-gray-500">Task not found</p>
            </div>
          ) : isEditing ? (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Title <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editFormData.title}
                  onChange={e => setEditFormData({ ...editFormData, title: e.target.value })}
                  disabled={isSaving}
                  maxLength={255}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
                  placeholder="Task title"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <textarea
                  value={editFormData.description}
                  onChange={e => setEditFormData({ ...editFormData, description: e.target.value })}
                  disabled={isSaving}
                  rows={4}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
                  placeholder="Task description"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <select
                  value={editFormData.priority}
                  onChange={e =>
                    setEditFormData({
                      ...editFormData,
                      priority: e.target.value as TaskPriority,
                    })
                  }
                  disabled={isSaving}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </div>

              <div className="flex gap-3 pt-4">
                <button
                  onClick={handleCancelEdit}
                  disabled={isSaving}
                  className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSave}
                  disabled={isSaving || !editFormData.title.trim()}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  {isSaving ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-6">
              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-500">Title</label>
                <p className="mt-1 text-gray-900 text-lg font-medium">{task.title}</p>
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-500">Description</label>
                <p className="mt-1 text-gray-700 whitespace-pre-wrap">
                  {task.description || 'No description'}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-500">Status</label>
                <p className="mt-1 text-gray-900 capitalize">{task.status.replace('_', ' ')}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-500">Priority</label>
                <div className="mt-1">{getPriorityBadge(task.priority)}</div>
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-500">Created</label>
                <p className="mt-1 text-gray-900 font-mono text-sm">
                  {new Date(task.created_at).toLocaleString()}
                </p>
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-500">Last Updated</label>
                <p className="mt-1 text-gray-900 font-mono text-sm">
                  {new Date(task.updated_at).toLocaleString()}
                </p>
              </div>

              {task.current_session_id && (
                <div className="col-span-2 pt-4 border-t border-gray-200">
                  <label className="block text-sm font-medium text-gray-500 mb-2">
                    Execution Status
                  </label>
                  <div className="bg-blue-50 rounded-lg p-3 flex items-center gap-2">
                    {isExecuting && (
                      <svg
                        className="w-4 h-4 animate-spin text-blue-600 flex-shrink-0"
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
                    )}
                    <div>
                      <p className="text-sm font-medium text-blue-900">
                        {isExecuting ? 'Execution in progress' : 'Last execution'}
                      </p>
                      <p className="text-xs text-blue-700 font-mono">
                        Session ID: {task.current_session_id}
                      </p>
                      {task.execution_duration_ms > 0 && (
                        <p className="text-xs text-blue-700">
                          Duration: {(task.execution_duration_ms / 1000).toFixed(2)}s
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {taskId && (
                <ExecutionOutputPanel
                  projectId={projectId}
                  taskId={taskId}
                  sessionId={sessionId}
                  isExecuting={isExecuting}
                />
              )}
            </div>
          )}
        </div>

        {!isEditing && task && !isLoading && (
          <div className="p-6 border-t border-gray-200">
            {!showDeleteConfirm ? (
              <button
                onClick={() => setShowDeleteConfirm(true)}
                className="w-full px-4 py-2 border border-red-500 text-red-600 rounded-lg hover:bg-red-50 flex items-center justify-center gap-2"
              >
                <svg
                  className="w-5 h-5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
                Delete Task
              </button>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-red-600 font-medium text-center">
                  Are you sure? This action cannot be undone.
                </p>
                <div className="flex gap-3">
                  <button
                    onClick={() => setShowDeleteConfirm(false)}
                    disabled={isDeleting}
                    className="flex-1 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleDelete}
                    disabled={isDeleting}
                    className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                  >
                    {isDeleting ? 'Deleting...' : 'Confirm Delete'}
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </aside>
    </>
  )
}
