import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/auth/callback" element={<OidcCallback />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/projects/:id" element={<ProjectDetailPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  )
}

function HomePage() {
  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          OpenCode Project Manager
        </h1>
        <p className="text-xl text-gray-600 mb-8">
          Manage your projects with AI-powered coding assistance
        </p>
        <a
          href="/login"
          className="inline-block px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
        >
          Get Started
        </a>
      </div>
    </div>
  )
}

function LoginPage() {
  return (
    <div className="min-h-screen bg-gray-100 flex items-center justify-center">
      <div className="bg-white p-8 rounded-lg shadow-md">
        <h2 className="text-2xl font-bold mb-6">Login</h2>
        <button className="w-full px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition">
          Login with Keycloak
        </button>
      </div>
    </div>
  )
}

function OidcCallback() {
  return <div>Loading...</div>
}

function ProjectsPage() {
  return (
    <div className="min-h-screen bg-gray-100 p-8">
      <h1 className="text-3xl font-bold mb-6">Projects</h1>
      <div className="bg-white p-6 rounded-lg shadow-md">
        <p>No projects yet. Create your first project!</p>
      </div>
    </div>
  )
}

function ProjectDetailPage() {
  return (
    <div className="min-h-screen bg-gray-100 p-8">
      <h1 className="text-3xl font-bold mb-6">Project Details</h1>
      <div className="bg-white p-6 rounded-lg shadow-md">
        <p>Project details will be displayed here.</p>
      </div>
    </div>
  )
}

export default App
