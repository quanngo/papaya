import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { listTenants } from '../api/tenants'
import { listClaims } from '../api/claims'
import { useAuth } from '../auth/AuthContext'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import { formatDateTime, formatMoney, formatNumber } from '../i18n/format'
import styles from './DashboardPage.module.css'

const RECENT = 6
const routingBadge = { auto_approved: ui.badgeSuccess, manual_review: ui.badgeWarning, rejected: ui.badgeDanger }

/** Admin home: key numbers, quick actions, latest config changes and claims. */
export default function DashboardPage() {
  const { t } = useTranslation()
  const { can } = useAuth()
  const [tenants, setTenants] = useState(null)
  const [claims, setClaims] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    let alive = true
    listTenants()
      .then((list) => alive && setTenants(list))
      .catch((err) => alive && setError(err))
    if (can('claims.read')) {
      listClaims('', 200)
        .then((list) => alive && setClaims(list))
        .catch(() => alive && setClaims([]))
    } else {
      setClaims([])
    }
    return () => {
      alive = false
    }
  }, [can])

  if (error) {
    return (
      <div className={ui.page}>
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error)}
        </p>
      </div>
    )
  }
  if (!tenants || !claims) return <Spinner label={t('common.loading')} />

  const count = (routing) => claims.filter((c) => c.routing === routing).length
  const auto = count('auto_approved')
  const recentChanges = [...tenants].sort((a, b) => b.updated_at.localeCompare(a.updated_at)).slice(0, RECENT)
  const stats = [
    { key: 'tenants', value: formatNumber(tenants.length), to: '/tenants' },
    { key: 'versions', value: formatNumber(tenants.reduce((n, tn) => n + tn.current_version, 0)), to: '/tenants' },
    { key: 'claims', value: formatNumber(claims.length), to: '/claims' },
    { key: 'manual', value: formatNumber(count('manual_review')), to: '/claims?routing=manual_review', tone: 'warning' },
    { key: 'autoRate', value: claims.length ? `${Math.round((auto / claims.length) * 100)}%` : '—', to: '/claims?routing=auto_approved', tone: 'success' },
  ]

  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <h1 className={ui.title}>{t('dashboard.title')}</h1>
          <p className={ui.subtitle}>{t('dashboard.subtitle')}</p>
        </div>
        <div className={ui.row}>
          {can('tenants.manage') && (
            <Link to="/tenants/new" className={`${ui.button} ${ui.primary}`}>
              + {t('tenants.new')}
            </Link>
          )}
          <Link to="/preview" className={ui.button}>
            ▶ {can('claims.process') ? t('dashboard.processClaim') : t('dashboard.previewClaim')}
          </Link>
          <Link to="/diff" className={ui.button}>
            ⇄ {t('layout.menu.diff')}
          </Link>
        </div>
      </div>

      <section className={styles.stats} aria-label={t('dashboard.stats')}>
        {stats.map((s) => (
          <Link key={s.key} to={s.to} className={`${styles.stat} ${s.tone ? styles[s.tone] : ''}`}>
            <span className={styles.statValue}>{s.value}</span>
            <span className={styles.statLabel}>{t(`dashboard.stat.${s.key}`)}</span>
          </Link>
        ))}
      </section>

      <div className={styles.columns}>
        <section className={ui.card} aria-labelledby="recent-changes">
          <div className={styles.cardHead}>
            <h2 id="recent-changes" className={ui.cardTitle}>
              {t('dashboard.recentChanges')}
            </h2>
            <Link to="/tenants">{t('dashboard.viewAll')}</Link>
          </div>
          {recentChanges.length === 0 ? (
            <p className={ui.empty}>{t('tenants.empty')}</p>
          ) : (
            <ul className={styles.list}>
              {recentChanges.map((tn) => (
                <li key={tn.id}>
                  <Link to={`/tenants/${tn.slug}`} className={styles.itemMain}>
                    {tn.name}
                  </Link>
                  <span className={`${ui.badge} ${ui.badgePrimary}`}>v{tn.current_version}</span>
                  <span className={styles.itemMeta}>{formatDateTime(tn.updated_at)}</span>
                  <Link to={`/tenants/${tn.slug}/history`} className={styles.itemLink}>
                    {t('tenants.history')}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>

        {can('claims.read') && (
          <section className={ui.card} aria-labelledby="recent-claims">
            <div className={styles.cardHead}>
              <h2 id="recent-claims" className={ui.cardTitle}>
                {t('dashboard.recentClaims')}
              </h2>
              <Link to="/claims">{t('dashboard.viewAll')}</Link>
            </div>
            {claims.length === 0 ? (
              <p className={ui.empty}>{t('claims.empty')}</p>
            ) : (
              <ul className={styles.list}>
                {claims.slice(0, RECENT).map((c) => (
                  <li key={c.id}>
                    <span className={styles.itemMain}>
                      {c.result?.tenant.name || c.tenant_slug} · {t(`claimTypes.${c.claim_type}`)} · {formatMoney(c.amount)}
                    </span>
                    <span className={`${ui.badge} ${routingBadge[c.routing]}`}>{t(`routing.${c.routing}`)}</span>
                    <span className={styles.itemMeta}>{formatDateTime(c.created_at)}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}
      </div>
    </div>
  )
}
