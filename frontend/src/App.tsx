import { BrowserRouter as Router, Routes, Route, Navigate, Link } from 'react-router-dom'

import { AuthProvider } from '@/contexts/AuthContext'
import { AppLayout } from '@/components/AppLayout'
import { LoginPage } from '@/pages/LoginPage'
import { OidcCallbackPage } from '@/pages/OidcCallbackPage'
import { ProjectDetailPage } from '@/pages/ProjectDetailPage'
import { ProtectedRoute } from '@/components/ProtectedRoute'
import { ProjectList } from '@/components/Projects/ProjectList'
import { KanbanBoard } from '@/components/Kanban/KanbanBoard'
import { useAuth } from '@/hooks/useAuth'
import { useParams } from 'react-router-dom'

function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/auth/callback" element={<OidcCallbackPage />} />
          <Route
            path="/projects"
            element={
              <ProtectedRoute>
                <AppLayout>
                  <ProjectList />
                </AppLayout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/projects/:id"
            element={
              <ProtectedRoute>
                <AppLayout>
                  <ProjectDetailPage />
                </AppLayout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/projects/:id/tasks"
            element={
              <ProtectedRoute>
                <AppLayout>
                  <KanbanBoardPage />
                </AppLayout>
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Router>
    </AuthProvider>
  )
}

function HomePage() {
  const { isAuthenticated } = useAuth()

  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">OpenCode Project Manager</h1>
        <p className="text-xl text-gray-600 mb-8">
          Manage your projects with AI-powered coding assistance
        </p>
        <div className="flex justify-center gap-4">
          {isAuthenticated ? (
            <Link
              to="/projects"
              className="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              Go to Projects
            </Link>
          ) : (
            <a
              href="/login"
              className="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              Get Started
            </a>
          )}
        </div>
      </div>
    </div>
  )
}

function KanbanBoardPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center">
        <div className="text-red-600">Project ID is missing</div>
      </div>
    )
  }

  return <KanbanBoard projectId={id} />
}

export default App
