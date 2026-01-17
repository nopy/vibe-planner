import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import type { Project } from '@/types'

interface ProjectCardProps {
  project: Project
  onDelete: (id: string) => Promise<void>
}

export function ProjectCard({ project, onDelete }: ProjectCardProps) {
  const navigate = useNavigate()
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const handleCardClick = () => {
    navigate(`/projects/${project.id}`)
  }

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    setShowDeleteConfirm(true)
  }

  const handleConfirmDelete = async (e: React.MouseEvent) => {
    e.stopPropagation()
    setIsDeleting(true)
    try {
      await onDelete(project.id)
    } catch (error) {
      console.error('Failed to delete project:', error)
      setIsDeleting(false)
      setShowDeleteConfirm(false)
    }
  }

  const handleCancelDelete = (e: React.MouseEvent) => {
    e.stopPropagation()
    setShowDeleteConfirm(false)
  }

  const getStatusBadge = () => {
    const baseClasses = 'px-2 py-1 text-xs font-medium rounded-full'

    switch (project.status) {
      case 'ready':
        return <span className={`${baseClasses} bg-green-100 text-green-800`}>Ready</span>
      case 'initializing':
        return <span className={`${baseClasses} bg-yellow-100 text-yellow-800`}>Initializing</span>
      case 'error':
        return <span className={`${baseClasses} bg-red-100 text-red-800`}>Error</span>
      case 'archived':
        return <span className={`${baseClasses} bg-gray-100 text-gray-800`}>Archived</span>
      default:
        return <span className={`${baseClasses} bg-gray-100 text-gray-800`}>Unknown</span>
    }
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  return (
    <div
      onClick={handleCardClick}
      className="bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow cursor-pointer p-6 relative"
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1">
          <h3 className="text-xl font-semibold text-gray-900 mb-1">{project.name}</h3>
          {project.description && (
            <p className="text-gray-600 text-sm line-clamp-2">{project.description}</p>
          )}
        </div>
        <div className="ml-4">{getStatusBadge()}</div>
      </div>

      <div className="flex items-center justify-between text-sm text-gray-500">
        <span>Created {formatDate(project.created_at)}</span>

        {!showDeleteConfirm ? (
          <button
            onClick={handleDeleteClick}
            className="text-red-600 hover:text-red-700 font-medium"
          >
            Delete
          </button>
        ) : (
          <div className="flex gap-2">
            <button
              onClick={handleConfirmDelete}
              disabled={isDeleting}
              className="px-3 py-1 bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
            >
              {isDeleting ? 'Deleting...' : 'Confirm'}
            </button>
            <button
              onClick={handleCancelDelete}
              disabled={isDeleting}
              className="px-3 py-1 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50"
            >
              Cancel
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
