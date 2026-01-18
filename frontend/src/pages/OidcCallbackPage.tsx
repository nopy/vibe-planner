import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

export function OidcCallbackPage() {
  const [searchParams] = useSearchParams()
  const { handleCallback } = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)
  const hasProcessed = useRef(false)

  useEffect(() => {
    const processCallback = async () => {
      // Prevent duplicate processing (React StrictMode double-invokes effects)
      if (hasProcessed.current) {
        return
      }

      const code = searchParams.get('code')
      const errorParam = searchParams.get('error')

      if (errorParam) {
        setError(`Authentication error: ${errorParam}`)
        return
      }

      if (!code) {
        setError('No authorization code received')
        return
      }

      hasProcessed.current = true

      try {
        await handleCallback(code)
        navigate('/projects')
      } catch (err) {
        setError('Failed to complete authentication')
        console.error('Callback error:', err)
      }
    }

    processCallback()
  }, [searchParams, handleCallback, navigate])

  if (error) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="bg-white p-8 rounded-lg shadow-md max-w-md w-full">
          <h2 className="text-2xl font-bold mb-4 text-red-600">Authentication Failed</h2>
          <p className="text-gray-700 mb-6">{error}</p>
          <button
            onClick={() => navigate('/login')}
            className="w-full px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
          >
            Try Again
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="text-center">
        <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
        <p className="text-gray-700 text-lg">Completing authentication...</p>
      </div>
    </div>
  )
}
