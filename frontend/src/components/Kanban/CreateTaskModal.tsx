import { useState } from 'react'

import { createTask } from '@/services/api'
import type { Task, CreateTaskRequest, TaskPriority } from '@/types'

interface CreateTaskModalProps {
  isOpen: boolean
  onClose: () => void
  onTaskCreated: (task: Task) => void
  projectId: string
}

export function CreateTaskModal({
  isOpen,
  onClose,
  onTaskCreated,
  projectId,
}: CreateTaskModalProps) {
  const [formData, setFormData] = useState<{
    title: string
    description: string
    priority: TaskPriority
  }>({
    title: '',
    description: '',
    priority: 'medium',
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [fieldErrors, setFieldErrors] = useState<{
    title?: string
    priority?: string
  }>({})

  const validateForm = (): boolean => {
    const errors: { title?: string; priority?: string } = {}

    if (!formData.title.trim()) {
      errors.title = 'Task title is required'
    } else if (formData.title.length > 255) {
      errors.title = 'Task title must be less than 255 characters'
    }

    if (formData.priority) {
      const validPriorities: TaskPriority[] = ['low', 'medium', 'high']
      if (!validPriorities.includes(formData.priority)) {
        errors.priority = 'Invalid priority selected'
      }
    }

    setFieldErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (!validateForm()) {
      return
    }

    setIsSubmitting(true)

    try {
      const payload: CreateTaskRequest = {
        title: formData.title.trim(),
        description: formData.description?.trim() || undefined,
        priority: formData.priority || 'medium',
      }

      const newTask = await createTask(projectId, payload)
      onTaskCreated(newTask)
      handleClose()
    } catch (err) {
      console.error('Failed to create task:', err)
      setError('Failed to create task. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleClose = () => {
    setFormData({ title: '', description: '', priority: 'medium' })
    setError(null)
    setFieldErrors({})
    setIsSubmitting(false)
    onClose()
  }

  if (!isOpen) return null

  return (
    <div
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      onClick={handleClose}
    >
      <div
        className="bg-white rounded-lg shadow-xl max-w-md w-full p-6"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-2xl font-bold text-gray-900">Create New Task</h2>
          <button
            onClick={handleClose}
            disabled={isSubmitting}
            className="text-gray-400 hover:text-gray-600 disabled:opacity-50"
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

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label htmlFor="title" className="block text-sm font-medium text-gray-700 mb-1">
              Task Title <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              id="title"
              value={formData.title}
              onChange={e => setFormData({ ...formData, title: e.target.value })}
              disabled={isSubmitting}
              className={`w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 ${
                fieldErrors.title ? 'border-red-500' : 'border-gray-300'
              }`}
              placeholder="Implement user authentication"
              maxLength={255}
            />
            {fieldErrors.title && <p className="mt-1 text-sm text-red-600">{fieldErrors.title}</p>}
          </div>

          <div className="mb-4">
            <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="description"
              value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })}
              disabled={isSubmitting}
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
              placeholder="Detailed description of the task..."
            />
          </div>

          <div className="mb-6">
            <label htmlFor="priority" className="block text-sm font-medium text-gray-700 mb-1">
              Priority
            </label>
            <select
              id="priority"
              value={formData.priority}
              onChange={e => setFormData({ ...formData, priority: e.target.value as TaskPriority })}
              disabled={isSubmitting}
              className={`w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 ${
                formData.priority === 'high'
                  ? 'text-red-600'
                  : formData.priority === 'medium'
                    ? 'text-yellow-600'
                    : 'text-green-600'
              }`}
            >
              <option value="low" className="text-green-600">
                Low
              </option>
              <option value="medium" className="text-yellow-600">
                Medium
              </option>
              <option value="high" className="text-red-600">
                High
              </option>
            </select>
          </div>

          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={handleClose}
              disabled={isSubmitting}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {isSubmitting ? 'Creating...' : 'Create Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
