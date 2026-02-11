import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './auth'
import Layout from './components/Layout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import Datasources from './pages/Datasources'
import Channels from './pages/Channels'
import Templates from './pages/Templates'
import Rules from './pages/Rules'
import Alerts from './pages/Alerts'
import Reports from './pages/Reports'
import Users from './pages/Users'
import Permissions from './pages/Permissions'

const ADMIN_ONLY_PATHS = ['/rules', '/datasources', '/channels', '/templates', '/users', '/permissions']

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RequireRoleRoute({ children }: { children: React.ReactNode }) {
  const { user, userLoading } = useAuth()
  const location = useLocation()
  const path = location.pathname
  const isAdminOnly = ADMIN_ONLY_PATHS.some((p) => path === p || path.startsWith(p + '/'))
  // Wait for /auth/me to finish before redirecting; otherwise refresh on /rules jumps to dashboard
  if (userLoading) return <>{children}</>
  if (isAdminOnly && user?.role !== 'admin') {
    return <Navigate to="/dashboard" replace state={{ from: path }} />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <PrivateRoute>
            <RequireRoleRoute>
              <Layout />
            </RequireRoleRoute>
          </PrivateRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="alerts" element={<Alerts />} />
        <Route path="reports" element={<Reports />} />
        <Route path="datasources" element={<Datasources />} />
        <Route path="channels" element={<Channels />} />
        <Route path="templates" element={<Templates />} />
        <Route path="rules" element={<Rules />} />
        <Route path="users" element={<Users />} />
        <Route path="permissions" element={<Permissions />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
