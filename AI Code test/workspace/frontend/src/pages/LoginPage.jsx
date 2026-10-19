import { useState } from 'react'
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { safeNextPath } from '../auth/ProtectedRoute'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import LanguageSwitcher from './LanguageSwitcher'
import styles from './LoginPage.module.css'

export default function LoginPage() {
  const { t } = useTranslation()
  const { user, loading, login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const next = safeNextPath(params.get('next'))
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  if (loading) return <Spinner label={t('common.loading')} />
  if (user) return <Navigate to={next} replace />

  const submit = async (e) => {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await login(email, password, remember)
      navigate(next, { replace: true })
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className={styles.wrap}>
      <div className={styles.lang}>
        <LanguageSwitcher />
      </div>
      <form className={`${ui.card} ${styles.card}`} onSubmit={submit} noValidate>
        <h1 className={ui.title}>{t('login.title')}</h1>
        <p className={ui.subtitle}>{t('login.subtitle')}</p>
        {error && (
          <p role="alert" className={ui.alert}>
            {actionErrorMessage(t, error, 'login')}
          </p>
        )}
        <label className={ui.field}>
          <span className={ui.label}>{t('login.email')}</span>
          <input type="email" autoComplete="username" required value={email} onChange={(e) => setEmail(e.target.value)} />
        </label>
        <label className={ui.field}>
          <span className={ui.label}>{t('login.password')}</span>
          <input type="password" autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>
        <label className={styles.remember}>
          <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} />
          {t('login.remember')}
        </label>
        <button type="submit" className={`${ui.button} ${ui.primary}`} disabled={busy || !email || !password}>
          {busy ? t('login.signingIn') : t('login.submit')}
        </button>
        <p className={ui.hint}>{t('login.demoHint')}</p>
      </form>
    </main>
  )
}
