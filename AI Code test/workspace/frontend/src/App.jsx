import { Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth/AuthContext'
import ProtectedRoute from './auth/ProtectedRoute'
import AppLayout from './pages/AppLayout'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import TenantsPage from './pages/TenantsPage'
import TenantEditPage from './pages/TenantEditPage'
import TenantHistoryPage from './pages/TenantHistoryPage'
import PreviewPage from './pages/PreviewPage'
import DiffPage from './pages/DiffPage'
import ClaimsPage from './pages/ClaimsPage'
import NotFoundPage from './pages/NotFoundPage'

/** The whole app without the router (main.jsx adds BrowserRouter, tests MemoryRouter). */
export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute />}>
          <Route element={<AppLayout />}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<ProtectedRoute permission="tenants.read"><DashboardPage /></ProtectedRoute>} />
            <Route path="/tenants" element={<ProtectedRoute permission="tenants.read"><TenantsPage /></ProtectedRoute>} />
            <Route path="/tenants/new" element={<ProtectedRoute permission="tenants.manage"><TenantEditPage /></ProtectedRoute>} />
            <Route path="/tenants/:id" element={<ProtectedRoute permission="tenants.read"><TenantEditPage /></ProtectedRoute>} />
            <Route path="/tenants/:id/history" element={<ProtectedRoute permission="tenants.read"><TenantHistoryPage /></ProtectedRoute>} />
            <Route path="/preview" element={<ProtectedRoute permission="tenants.read"><PreviewPage /></ProtectedRoute>} />
            <Route path="/diff" element={<ProtectedRoute permission="tenants.read"><DiffPage /></ProtectedRoute>} />
            <Route path="/claims" element={<ProtectedRoute permission="claims.read"><ClaimsPage /></ProtectedRoute>} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
        </Route>
      </Routes>
    </AuthProvider>
  )
}
