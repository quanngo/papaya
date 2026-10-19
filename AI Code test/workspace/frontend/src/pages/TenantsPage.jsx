import { useEffect, useMemo, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getTenant, listTenants } from '../api/tenants'
import { useAuth } from '../auth/AuthContext'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import TenantLogo from '../components/TenantLogo'
import ui from '../components/ui.module.css'
import { formatDateTime, formatMoney } from '../i18n/format'
import styles from './TenantsPage.module.css'

/** Config summary shown in the list, so admins compare insurers without opening each one. */
function summarize(config) {
  const channels = new Set()
  for (const rule of Object.values(config.notifications.events || {})) if (rule.enabled) rule.channels.forEach((c) => channels.add(c))
  return {
    claimTypes: config.claim_types.enabled,
    threshold: config.approval.auto_approval_threshold,
    tiers: config.approval.tiers.length,
    channels: [...channels],
    customFields: config.custom_fields?.length || 0,
    color: config.branding.primary_color,
    logoUrl: config.branding.logo_url,
  }
}

export default function TenantsPage() {
  const { t } = useTranslation()
  const { can } = useAuth()
  const location = useLocation()
  const [params, setParams] = useSearchParams()
  const query = params.get('q') || ''
  const sort = params.get('sort') || 'name'
  const showArchived = params.get('archived') === '1'
  const [tenants, setTenants] = useState(null)
  const [summaries, setSummaries] = useState({})
  const [error, setError] = useState(null)

  const setParam = (key, value) => {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  useEffect(() => {
    let alive = true
    setTenants(null)
    setError(null)
    listTenants(showArchived)
      .then((list) => {
        if (!alive) return
        setTenants(list)
        // Summaries are a progressive enhancement: the table renders before they arrive.
        Promise.all(list.map((tn) => getTenant(tn.id).catch(() => null))).then((details) => {
          if (!alive) return
          setSummaries(Object.fromEntries(details.filter(Boolean).map((d) => [d.id, summarize(d.config)])))
        })
      })
      .catch((err) => alive && setError(err))
    return () => {
      alive = false
    }
  }, [showArchived])

  const rows = useMemo(() => {
    if (!tenants) return []
    const q = query.trim().toLowerCase()
    const filtered = q ? tenants.filter((tn) => tn.name.toLowerCase().includes(q) || tn.slug.includes(q)) : tenants
    const sorted = [...filtered]
    if (sort === 'updated') sorted.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
    else if (sort === 'version') sorted.sort((a, b) => b.current_version - a.current_version)
    else sorted.sort((a, b) => a.name.localeCompare(b.name))
    return sorted
  }, [tenants, query, sort])

  const canManage = can('tenants.manage')

  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <h1 className={ui.title}>{t('tenants.title')}</h1>
          <p className={ui.subtitle}>{t('tenants.subtitle')}</p>
        </div>
        {canManage && (
          <Link to="/tenants/new" className={`${ui.button} ${ui.primary}`}>
            + {t('tenants.new')}
          </Link>
        )}
      </div>

      {location.state?.archived && (
        <p className={ui.success} role="status">
          {t('tenants.archivedNotice', { name: location.state.archived })}
        </p>
      )}

      <div className={styles.toolbar}>
        <input type="search" className={styles.search} aria-label={t('tenants.search')} placeholder={t('tenants.searchPlaceholder')} value={query} onChange={(e) => setParam('q', e.target.value)} />
        <select className={styles.sort} aria-label={t('tenants.sortBy')} value={sort} onChange={(e) => setParam('sort', e.target.value === 'name' ? '' : e.target.value)}>
          <option value="name">{t('tenants.sort.name')}</option>
          <option value="updated">{t('tenants.sort.updated')}</option>
          <option value="version">{t('tenants.sort.version')}</option>
        </select>
        <label className={ui.row}>
          <input type="checkbox" checked={showArchived} onChange={(e) => setParam('archived', e.target.checked ? '1' : '')} />
          {t('tenants.showArchived')}
        </label>
        {tenants && <span className={`${ui.muted} ${styles.count}`}>{t('tenants.count', { shown: rows.length, total: tenants.length })}</span>}
      </div>

      {error && (
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error)}
        </p>
      )}
      {!tenants && !error && <Spinner label={t('common.loading')} />}

      {tenants && (
        <div className={`${ui.card} ${ui.tableWrap} ${styles.tableCard}`}>
          {rows.length === 0 ? (
            <p className={ui.empty}>{query ? t('tenants.noMatch', { q: query }) : t('tenants.empty')}</p>
          ) : (
            <table className={`${ui.table} ${styles.table}`}>
              <thead>
                <tr>
                  <th>{t('tenants.name')}</th>
                  <th>{t('tenants.claimTypes')}</th>
                  <th className={styles.num}>{t('tenants.threshold')}</th>
                  <th className={styles.num}>{t('tenants.tiers')}</th>
                  <th>{t('tenants.channels')}</th>
                  <th>{t('tenants.version')}</th>
                  <th>{t('tenants.updated')}</th>
                  <th>
                    <span className="visually-hidden">{t('common.actions')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {rows.map((tn) => {
                  const s = summaries[tn.id]
                  return (
                    <tr key={tn.id}>
                      <td>
                        <div className={styles.nameCell}>
                          <TenantLogo url={s?.logoUrl} name={tn.name} color={s?.color} size="sm" />
                          <div>
                            <Link to={`/tenants/${tn.slug}`} className={styles.name}>
                              {tn.name}
                            </Link>{' '}
                            {tn.archived_at && <span className={ui.badge}>{t('tenants.archived')}</span>}
                            <div className={ui.mono}>{tn.slug}</div>
                          </div>
                        </div>
                      </td>
                      <td>
                        {s ? (
                          <div className={styles.tags}>
                            {s.claimTypes.map((ct) => (
                              <span key={ct} className={ui.badge} title={t(`claimTypes.${ct}`)}>
                                {t(`claimTypes.${ct}`)}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className={ui.muted}>…</span>
                        )}
                      </td>
                      <td className={styles.num}>{s ? formatMoney(s.threshold) : '…'}</td>
                      <td className={styles.num}>{s ? s.tiers : '…'}</td>
                      <td className={styles.small}>{s ? s.channels.map((c) => t(`channels.${c}`)).join(', ') || '—' : '…'}</td>
                      <td>
                        <span className={`${ui.badge} ${ui.badgePrimary}`}>v{tn.current_version}</span>
                      </td>
                      <td className={`${ui.muted} ${styles.small}`}>{formatDateTime(tn.updated_at)}</td>
                      <td>
                        <div className={styles.actions}>
                          <Link to={`/tenants/${tn.slug}`} className={`${ui.button} ${ui.small}`}>
                            {canManage && !tn.archived_at ? t('common.edit') : t('common.view')}
                          </Link>
                          <Link to={`/tenants/${tn.slug}/history`} className={`${ui.button} ${ui.small}`} title={t('tenants.history')}>
                            {t('tenants.history')}
                          </Link>
                          {!tn.archived_at && (
                            <Link to={`/preview?tenant=${tn.slug}`} className={`${ui.button} ${ui.small}`} aria-label={t('tenants.previewFor', { name: tn.name })} title={t('layout.menu.preview')}>
                              ▶
                            </Link>
                          )}
                          <Link to={`/diff?a=${tn.slug}`} className={`${ui.button} ${ui.small}`} aria-label={t('tenants.compareFor', { name: tn.name })} title={t('tenant.compare')}>
                            ⇄
                          </Link>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  )
}
