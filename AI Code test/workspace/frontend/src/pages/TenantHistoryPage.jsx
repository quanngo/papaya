import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { diffTenants, getTenant, listVersions, rollbackTenant } from '../api/tenants'
import { useAuth } from '../auth/AuthContext'
import { actionErrorMessage } from '../auth/errorMessages'
import ConfirmDialog from '../components/ConfirmDialog'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import { formatDateTime } from '../i18n/format'
import ChangesTable from '../tenant/ChangesTable'
import styles from './TenantHistoryPage.module.css'

const sourceBadge = { create: ui.badgePrimary, update: '', rollback: ui.badgeWarning }

export default function TenantHistoryPage() {
  const { id } = useParams()
  const { t } = useTranslation()
  const { can } = useAuth()
  const [tenant, setTenant] = useState(null)
  const [versions, setVersions] = useState(null)
  const [selected, setSelected] = useState(null)
  const [diff, setDiff] = useState(null)
  const [error, setError] = useState(null)
  const [confirm, setConfirm] = useState(null)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      const [tn, list] = await Promise.all([getTenant(id), listVersions(id)])
      setTenant(tn)
      setVersions(list)
      return list
    } catch (err) {
      setError(err)
      return null
    }
  }, [id])

  useEffect(() => {
    load().then((list) => {
      const first = list?.find((v) => !v.is_current)
      if (first) setSelected(first.version)
    })
  }, [load])

  useEffect(() => {
    if (!selected || !tenant) return undefined
    let alive = true
    setDiff(null)
    diffTenants(`${tenant.id}@${selected}`, tenant.id)
      .then((d) => alive && setDiff(d))
      .catch((err) => alive && setError(err))
    return () => {
      alive = false
    }
  }, [selected, tenant])

  const rollback = async () => {
    setBusy(true)
    try {
      const updated = await rollbackTenant(tenant.id, confirm, '')
      setNotice(t('history.rolledBack', { from: confirm, n: updated.current_version }))
      setConfirm(null)
      await load()
      setSelected(confirm)
    } catch (err) {
      setError(err)
      setConfirm(null)
    } finally {
      setBusy(false)
    }
  }

  if (error && !versions) {
    return (
      <div className={ui.page}>
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error, 'tenant')}
        </p>
      </div>
    )
  }
  if (!versions || !tenant) return <Spinner label={t('common.loading')} />

  const canRollback = can('tenants.manage') && !tenant.archived_at
  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <p className={ui.muted}>
            <Link to={`/tenants/${tenant.slug}`}>← {tenant.name}</Link>
          </p>
          <h1 className={ui.title}>{t('history.title', { name: tenant.name })}</h1>
          <p className={ui.subtitle}>{t('history.subtitle')}</p>
        </div>
      </div>
      {notice && (
        <p className={ui.success} role="status">
          {notice}
        </p>
      )}
      {error && (
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error, 'tenant')}
        </p>
      )}

      <div className={styles.layout}>
        <ol className={`${ui.card} ${styles.timeline}`} aria-label={t('history.versions')}>
          {versions.map((v) => (
            <li key={v.version} className={`${styles.item} ${selected === v.version ? styles.selected : ''}`}>
              <button type="button" className={styles.pick} onClick={() => setSelected(v.version)} aria-pressed={selected === v.version} disabled={v.is_current}>
                <span className={styles.top}>
                  <strong>v{v.version}</strong>
                  <span className={`${ui.badge} ${sourceBadge[v.source]}`}>{t(`history.source.${v.source}`, { n: v.rolled_back_from })}</span>
                  {v.is_current && <span className={`${ui.badge} ${ui.badgeSuccess}`}>{t('history.current')}</span>}
                </span>
                <span className={styles.note}>{v.note || <em className={ui.muted}>{t('history.noNote')}</em>}</span>
                <span className={ui.hint}>
                  {formatDateTime(v.created_at)} · {v.created_by_email || t('history.system')}
                </span>
              </button>
            </li>
          ))}
        </ol>

        <section className={ui.card} aria-live="polite">
          {!selected && <p className={ui.muted}>{t('history.onlyOne')}</p>}
          {selected && (
            <>
              <div className={ui.header}>
                <h2 className={ui.cardTitle}>{t('history.diffTitle', { v: selected, current: tenant.current_version })}</h2>
                {canRollback && (
                  <button type="button" className={`${ui.button} ${ui.primary}`} onClick={() => setConfirm(selected)}>
                    {t('history.rollbackTo', { v: selected })}
                  </button>
                )}
              </div>
              {!diff ? <Spinner label={t('common.loading')} /> : <ChangesTable changes={diff.changes} leftLabel={`v${selected}`} rightLabel={t('history.currentN', { n: tenant.current_version })} />}
            </>
          )}
        </section>
      </div>

      <ConfirmDialog
        open={confirm !== null}
        title={t('history.rollbackTitle')}
        message={t('history.rollbackMessage', { v: confirm, n: tenant.current_version + 1 })}
        confirmLabel={t('history.rollbackTo', { v: confirm })}
        busy={busy}
        onConfirm={rollback}
        onCancel={() => setConfirm(null)}
      />
    </div>
  )
}
