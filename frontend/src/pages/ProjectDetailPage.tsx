import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'

import { useProjectStatus } from '@/hooks/useProjectStatus'
import { deleteProject, getProject } from '@/services/api'
import type { Project } from '@/types'

export function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [project, setProject] = useState<Project | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)

  const {
    status: liveStatus,
    isConnected,
    error: wsError,
    reconnect,
  } = useProjectStatus(id || '')

  useEffect(() => {
    const fetchProject = async () => {
      if (!id) {
        setError('Project ID is missing')
        setIsLoading(false)
        return
      }

      setIsLoading(true)
      setError(null)

      try {
        const data = await getProject(id)
        setProject(data)
      } catch (err) {
        console.error('Failed to fetch project:', err)
        setError('Failed to load project. Please try again.')
      } finally {
        setIsLoading(false)
      }
    }

    fetchProject()
  }, [id])

  useEffect(() => {
    if (liveStatus && project) {
      setProject((prev) => (prev ? { ...prev, pod_status: liveStatus } : null))
    }
  }, [liveStatus, project])

  const handleDelete = async () => {
    if (!id) return

    setIsDeleting(true)
    try {
      await deleteProject(id)
      navigate('/projects')
    } catch (err) {
      console.error('Failed to delete project:', err)
      setError('Failed to delete project. Please try again.')
      setIsDeleting(false)
      setShowDeleteConfirm(false)
    }
  }

  const getStatusBadge = (status: string) => {
    const baseClasses = 'px-3 py-1 text-sm font-medium rounded-full'

    switch (status) {
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
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-100">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex items-center justify-center py-12">
            <div className="text-center">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading project...</p>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (error || !project) {
    return (
      <div className="min-h-screen bg-gray-100">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <p className="text-red-600">{error || 'Project not found'}</p>
            <button
              onClick={() => navigate('/projects')}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Back to Projects
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-100">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <nav className="mb-6">
          <button
            onClick={() => navigate('/projects')}
            className="text-blue-600 hover:text-blue-700 flex items-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
            Back to Projects
          </button>
        </nav>

        <div className="bg-white rounded-lg shadow-md p-8">
          <div className="flex items-start justify-between mb-6">
            <div className="flex-1">
              <h1 className="text-3xl font-bold text-gray-900 mb-2">{project.name}</h1>
              {project.description && <p className="text-gray-600">{project.description}</p>}
            </div>
            <div className="flex flex-col items-end gap-2">
              {getStatusBadge(project.status)}
              <div className="flex items-center gap-2 text-xs">
                <div
                  className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}
                ></div>
                <span className="text-gray-500">
                  {isConnected ? 'Live updates' : 'Disconnected'}
                </span>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            <div>
              <h2 className="text-sm font-medium text-gray-500 mb-1">Project ID</h2>
              <p className="text-gray-900 font-mono text-sm">{project.id}</p>
            </div>

            <div>
              <h2 className="text-sm font-medium text-gray-500 mb-1">Slug</h2>
              <p className="text-gray-900 font-mono text-sm">{project.slug}</p>
            </div>

            {project.pod_name && (
              <div>
                <h2 className="text-sm font-medium text-gray-500 mb-1">Pod Name</h2>
                <p className="text-gray-900 font-mono text-sm">{project.pod_name}</p>
              </div>
            )}

            {project.pod_namespace && (
              <div>
                <h2 className="text-sm font-medium text-gray-500 mb-1">Namespace</h2>
                <p className="text-gray-900 font-mono text-sm">{project.pod_namespace}</p>
              </div>
            )}

            {project.workspace_pvc_name && (
              <div>
                <h2 className="text-sm font-medium text-gray-500 mb-1">PVC Name</h2>
                <p className="text-gray-900 font-mono text-sm">{project.workspace_pvc_name}</p>
              </div>
            )}

            {project.pod_status && (
              <div>
                <h2 className="text-sm font-medium text-gray-500 mb-1">Pod Status</h2>
                <div className="flex items-center gap-2">
                  <p className="text-gray-900">{project.pod_status}</p>
                  {isConnected && (
                    <span className="inline-flex items-center text-xs text-green-600">
                      <svg className="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fillRule="evenodd"
                          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                          clipRule="evenodd"
                        />
                      </svg>
                      Live
                    </span>
                  )}
                </div>
              </div>
            )}

            <div>
              <h2 className="text-sm font-medium text-gray-500 mb-1">Created</h2>
              <p className="text-gray-900">{formatDate(project.created_at)}</p>
            </div>

            <div>
              <h2 className="text-sm font-medium text-gray-500 mb-1">Last Updated</h2>
              <p className="text-gray-900">{formatDate(project.updated_at)}</p>
            </div>
          </div>

          <div className="border-t pt-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Project Actions</h2>

            {wsError && (
              <div className="mb-4 bg-yellow-50 border border-yellow-200 rounded-lg p-4 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <svg className="w-5 h-5 text-yellow-600" fill="currentColor" viewBox="0 0 20 20">
                    <path
                      fillRule="evenodd"
                      d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className="text-sm text-yellow-800">{wsError}</span>
                </div>
                <button
                  onClick={reconnect}
                  className="px-3 py-1 text-sm bg-yellow-600 text-white rounded hover:bg-yellow-700"
                >
                  Reconnect
                </button>
              </div>
            )}

            <div className="flex gap-4">
              {!showDeleteConfirm ? (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
                >
                  Delete Project
                </button>
              ) : (
                <div className="flex gap-3">
                  <button
                    onClick={handleDelete}
                    disabled={isDeleting}
                    className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                  >
                    {isDeleting ? 'Deleting...' : 'Confirm Delete'}
                  </button>
                  <button
                    onClick={() => setShowDeleteConfirm(false)}
                    disabled={isDeleting}
                    className="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 disabled:opacity-50"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>

            {showDeleteConfirm && (
              <p className="mt-3 text-sm text-red-600">
                Warning: This will delete the project and all associated resources (Kubernetes pod
                and PVC). This action cannot be undone.
              </p>
            )}
          </div>

          <div className="border-t mt-6 pt-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Coming Soon</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="border border-gray-200 rounded-lg p-4 text-center">
                <p className="text-gray-500">Tasks</p>
              </div>
              <div className="border border-gray-200 rounded-lg p-4 text-center">
                <p className="text-gray-500">Files</p>
              </div>
              <div className="border border-gray-200 rounded-lg p-4 text-center">
                <p className="text-gray-500">Configuration</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
