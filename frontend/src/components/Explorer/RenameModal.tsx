import { useState, useEffect } from 'react'

interface RenameModalProps {
  isOpen: boolean
  onClose: () => void
  onRenamed: (oldPath: string, newPath: string) => void
  currentPath: string | null
}

export function RenameModal({ isOpen, onClose, onRenamed, currentPath }: RenameModalProps) {
  const [newName, setNewName] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isOpen && currentPath) {
      const basename = currentPath.split('/').pop() || ''
      setNewName(basename)
    }
  }, [isOpen, currentPath])

  const validateName = (name: string): string | null => {
    if (!name.trim()) {
      return 'Name is required'
    }
    if (name.includes('/') || name.includes('\\')) {
      return 'Name cannot contain slashes'
    }
    if (!/^[a-zA-Z0-9\-_.]+$/.test(name)) {
      return 'Name can only contain letters, numbers, dots, hyphens, and underscores'
    }
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentPath) return
    
    setError(null)

    const validationError = validateName(newName)
    if (validationError) {
      setError(validationError)
      return
    }

    const isImplemented = false
    if (isImplemented) {
        onRenamed(currentPath || '', newName)
    }

    setError('Rename functionality will be available in Phase 4.10')
  }

  const handleClose = () => {
    setNewName('')
    setError(null)
    setIsSubmitting(false)
    onClose()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-sm w-full p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold text-gray-900">Rename</h2>
          <button
            onClick={handleClose}
            disabled={isSubmitting}
            className="text-gray-400 hover:text-gray-600 disabled:opacity-50"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
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
            <label htmlFor="newName" className="block text-sm font-medium text-gray-700 mb-1">
              New Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              id="newName"
              value={newName}
              onChange={e => setNewName(e.target.value)}
              disabled={isSubmitting}
              className={`w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100 ${
                error ? 'border-red-500' : 'border-gray-300'
              }`}
              autoFocus
            />
            {currentPath && (
              <p className="mt-1 text-xs text-gray-500 truncate">
                Renaming: <span className="font-mono">{currentPath}</span>
              </p>
            )}
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
              {isSubmitting ? 'Renaming...' : 'Rename'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
