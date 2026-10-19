import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import ConfirmDialog from '../components/ConfirmDialog'
import ui from '../components/ui.module.css'
import LanguageSwitcher from './LanguageSwitcher'
import styles from './AppLayout.module.css'

// Grouped the way an admin works: overview first, then configuring insurers, then day-to-day claim operations.
const MENU = [
  { group: null, items: [{ to: '/dashboard', key: 'dashboard', icon: '◧', permission: 'tenants.read' }] },
  {
    group: 'config',
    items: [
      { to: '/tenants', key: 'tenants', icon: '🏢', permission: 'tenants.read' },
      { to: '/diff', key: 'diff', icon: '⇄', permission: 'tenants.read' },
    ],
  },
  {
    group: 'operations',
    items: [
      { to: '/preview', key: 'preview', icon: '▶', permission: 'tenants.read' },
      { to: '/claims', key: 'claims', icon: '☰', permission: 'claims.read' },
    ],
  },
]

/** Signed-in shell: main menu in a left sidebar (a top bar on phones). */
export default function AppLayout() {
  const { t } = useTranslation()
  const { user, can, logout } = useAuth()
  const navigate = useNavigate()
  const [confirmLogout, setConfirmLogout] = useState(false)
  const [busy, setBusy] = useState(false)

  const doLogout = async () => {
    setBusy(true)
    try {
      await logout()
    } finally {
      setBusy(false)
      setConfirmLogout(false)
      navigate('/login', { replace: true })
    }
  }

  return (
    <div className={styles.shell}>
      <aside className={styles.sidebar}>
        <NavLink to="/dashboard" className={styles.brand}>
          <span className={styles.logo} aria-hidden="true">
            ✓
          </span>
          {t('layout.appName')}
        </NavLink>

        <nav aria-label={t('layout.mainMenu')} className={styles.nav}>
          {MENU.map(({ group, items }) => {
            const visible = items.filter((m) => can(m.permission))
            if (!visible.length) return null
            return (
              <div key={group || 'top'} className={styles.group}>
                {group && <span className={styles.groupLabel}>{t(`layout.groups.${group}`)}</span>}
                {visible.map((m) => (
                  <NavLink key={m.to} to={m.to} className={({ isActive }) => `${styles.link} ${isActive ? styles.active : ''}`}>
                    <span className={styles.icon} aria-hidden="true">
                      {m.icon}
                    </span>
                    {t(`layout.menu.${m.key}`)}
                  </NavLink>
                ))}
              </div>
            )
          })}
        </nav>

        <div className={styles.footer}>
          <LanguageSwitcher className={styles.lang} />
          <div className={styles.who} title={user.email}>
            <span className={styles.name}>{user.display_name}</span>
            <span className={ui.badge}>{user.role.name}</span>
          </div>
          <button type="button" className={`${ui.button} ${ui.small} ${styles.logout}`} onClick={() => setConfirmLogout(true)}>
            {t('layout.logout')}
          </button>
        </div>
      </aside>

      <main className={styles.main}>
        <Outlet />
      </main>

      <ConfirmDialog
        open={confirmLogout}
        title={t('layout.logoutTitle')}
        message={t('layout.logoutMessage')}
        confirmLabel={t('layout.logout')}
        busy={busy}
        onConfirm={doLogout}
        onCancel={() => setConfirmLogout(false)}
      />
    </div>
  )
}
