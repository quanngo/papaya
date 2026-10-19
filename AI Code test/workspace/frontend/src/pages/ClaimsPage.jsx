import { Fragment, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { listClaims } from '../api/claims'
import { listTenants } from '../api/tenants'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import { formatDateTime, formatMoney } from '../i18n/format'
import ResultCard from '../tenant/ResultCard'

const routingBadge = { auto_approved: ui.badgeSuccess, manual_review: ui.badgeWarning, rejected: ui.badgeDanger }

/** Claims stored by processClaim, newest first. */
const ROUTINGS = ['auto_approved', 'manual_review', 'rejected']

export default function ClaimsPage() {
  const { t } = useTranslation()
  const [params, setParams] = useSearchParams()
  const routing = ROUTINGS.includes(params.get('routing')) ? params.get('routing') : ''
  const [tenants, setTenants] = useState([])
  const [tenantId, setTenantId] = useState('')
  const setRouting = (value) => {
    const next = new URLSearchParams(params)
    if (value) next.set('routing', value)
    else next.delete('routing')
    setParams(next, { replace: true })
  }
  const [claims, setClaims] = useState(null)
  const [open, setOpen] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    listTenants(true).then(setTenants).catch(() => {})
  }, [])

  useEffect(() => {
    let alive = true
    setClaims(null)
    setError(null)
    listClaims(tenantId)
      .then((list) => alive && setClaims(list))
      .catch((err) => alive && setError(err))
    return () => {
      alive = false
    }
  }, [tenantId])

  const visible = claims && (routing ? claims.filter((c) => c.routing === routing) : claims)

  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <h1 className={ui.title}>{t('claims.title')}</h1>
          <p className={ui.subtitle}>{t('claims.subtitle')}</p>
        </div>
        <select aria-label={t('claims.filterTenant')} value={tenantId} onChange={(e) => setTenantId(e.target.value)} style={{ width: 'auto' }}>
          <option value="">{t('claims.allTenants')}</option>
          {tenants.map((tn) => (
            <option key={tn.id} value={tn.id}>
              {tn.name}
            </option>
          ))}
        </select>
      </div>
      {error && (
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error)}
        </p>
      )}
      {!claims && !error && <Spinner label={t('common.loading')} />}
      {claims && (
        <div className={ui.row} role="group" aria-label={t('claims.filterRouting')} style={{ marginBottom: 12 }}>
          {['', ...ROUTINGS].map((r) => (
            <button
              key={r || 'all'}
              type="button"
              aria-pressed={routing === r}
              className={`${ui.button} ${ui.small} ${routing === r ? ui.primary : ''}`}
              onClick={() => setRouting(r)}
            >
              {r ? t(`routing.${r}`) : t('claims.allRoutings')} ({r ? claims.filter((c) => c.routing === r).length : claims.length})
            </button>
          ))}
        </div>
      )}
      {visible && (
        <div className={`${ui.card} ${ui.tableWrap}`}>
          {visible.length === 0 ? (
            <p className={ui.empty}>{t('claims.empty')}</p>
          ) : (
            <table className={ui.table}>
              <thead>
                <tr>
                  <th>{t('claims.when')}</th>
                  <th>{t('claims.tenant')}</th>
                  <th>{t('claim.claimType')}</th>
                  <th>{t('claim.amount')}</th>
                  <th>{t('claims.routing')}</th>
                  <th>{t('result.sla')}</th>
                  <th>
                    <span className="visually-hidden">{t('common.actions')}</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {visible.map((c) => (
                  <Fragment key={c.id}>
                    <tr>
                      <td className={ui.muted}>{formatDateTime(c.created_at)}</td>
                      <td>
                        {c.result?.tenant.name || c.tenant_slug} <span className={ui.muted}>v{c.config_version}</span>
                      </td>
                      <td>{t(`claimTypes.${c.claim_type}`)}</td>
                      <td>{formatMoney(c.amount)}</td>
                      <td>
                        <span className={`${ui.badge} ${routingBadge[c.routing]}`}>{t(`routing.${c.routing}`)}</span> {c.approver_role}
                      </td>
                      <td>{c.sla_deadline ? formatDateTime(c.sla_deadline, c.result?.sla?.timezone) : '—'}</td>
                      <td>
                        <button type="button" className={`${ui.button} ${ui.small}`} aria-expanded={open === c.id} onClick={() => setOpen(open === c.id ? null : c.id)}>
                          {open === c.id ? t('common.hide') : t('common.details')}
                        </button>
                      </td>
                    </tr>
                    {open === c.id && c.result && (
                      <tr>
                        <td colSpan={7}>
                          <ResultCard result={c.result} stored />
                        </td>
                      </tr>
                    )}
                  </Fragment>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  )
}
