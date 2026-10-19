import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from './AuthContext'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'

/** Renders children only for a signed-in user holding `permission` (when given). */
export default function ProtectedRoute({ permission, children }) {
  const { user, loading, can } = useAuth()
  const location = useLocation()
  const { t } = useTranslation()

  if (loading) return <Spinner label={t('common.loading')} />
  if (!user) {
    const next = location.pathname + location.search
    return <Navigate to={`/login?next=${encodeURIComponent(next)}`} replace />
  }
  if (permission && !can(permission)) {
    return (
      <div className={ui.page}>
        <p role="alert" className={ui.alert}>
          {t('errors.common.FORBIDDEN')}
        </p>
      </div>
    )
  }
  return children ?? <Outlet />
}

/** Only same-app relative paths are allowed after login (no open redirect). */
export function safeNextPath(next) {
  if (typeof next !== 'string' || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/dashboard'
  return next
}
