import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { fetchDefaults, fetchMeta, validateConfig } from '../api/configs'
import { archiveTenant, createTenant, getTenant, updateTenant } from '../api/tenants'
import { previewClaim } from '../api/claims'
import { useAuth } from '../auth/AuthContext'
import { actionErrorMessage } from '../auth/errorMessages'
import ConfirmDialog from '../components/ConfirmDialog'
import Spinner from '../components/Spinner'
import TenantLogo from '../components/TenantLogo'
import ui from '../components/ui.module.css'
import ClaimForm, { emptyClaim, toClaim } from '../tenant/ClaimForm'
import ResultCard from '../tenant/ResultCard'
import { normalizeConfig, setIn } from '../tenant/configState'
import { SECTIONS } from './tenant-sections'
import styles from './TenantEditPage.module.css'

const VALIDATE_DELAY_MS = 400

/**
 * Route entry: a fresh form per tenant. Without the key, going from
 * /tenants/new to /tenants/<slug> after creating would reuse the old state.
 */
export default function TenantEditRoute() {
  const { id } = useParams()
  return <TenantEditPage key={id || 'new'} />
}

/** Create (no :id) or edit a tenant; read-only without tenants.manage. */
function TenantEditPage() {
  const { id } = useParams()
  const isNew = !id
  const { t, i18n } = useTranslation()
  const { can } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const canManage = can('tenants.manage')

  const [meta, setMeta] = useState(null)
  const [tenant, setTenant] = useState(null)
  const [config, setConfig] = useState(null)
  const [slug, setSlug] = useState('')
  const [note, setNote] = useState('')
  const [loadError, setLoadError] = useState(null)
  const [issues, setIssues] = useState([])
  const [validating, setValidating] = useState(false)
  const [saveError, setSaveError] = useState(null)
  const [saving, setSaving] = useState(false)
  const [savedVersion, setSavedVersion] = useState(null)
  const [dirty, setDirty] = useState(false)
  const [confirmArchive, setConfirmArchive] = useState(false)
  const [claim, setClaim] = useState(emptyClaim())
  const [preview, setPreview] = useState(null)
  const [previewError, setPreviewError] = useState(null)
  const [previewing, setPreviewing] = useState(false)
  const validateSeq = useRef(0)
  const formRef = useRef(null)
  const canSaveRef = useRef(false)

  // Ctrl+S / ⌘S saves, like a desktop admin tool.
  useEffect(() => {
    const onKey = (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
        e.preventDefault()
        if (canSaveRef.current) formRef.current?.requestSubmit()
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  // Warn before closing or reloading the tab with unsaved changes.
  useEffect(() => {
    if (!dirty) return undefined
    const onBeforeUnload = (e) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])

  const load = useCallback(async () => {
    setLoadError(null)
    try {
      const [m, data] = await Promise.all([fetchMeta(), isNew ? fetchDefaults() : getTenant(id)])
      setMeta(m)
      if (isNew) {
        setTenant(null)
        setConfig(normalizeConfig(data))
        setSlug('')
      } else {
        setTenant(data)
        setConfig(normalizeConfig(data.config))
        setSlug(data.slug)
      }
      setNote('')
      setDirty(false)
    } catch (err) {
      setLoadError(err)
    }
  }, [id, isNew])

  useEffect(() => {
    setConfig(null)
    setSavedVersion(null)
    setPreview(null)
    load()
  }, [load])

  // Live validation by the backend (single source of truth). Re-runs on
  // language change so messages follow the UI language.
  useEffect(() => {
    if (!config || !canManage) return undefined
    const seq = ++validateSeq.current
    const controller = new AbortController()
    setValidating(true)
    const timer = setTimeout(() => {
      validateConfig(config, controller.signal)
        .then((r) => seq === validateSeq.current && setIssues(r.issues || []))
        .catch(() => {})
        .finally(() => seq === validateSeq.current && setValidating(false))
    }, VALIDATE_DELAY_MS)
    return () => {
      clearTimeout(timer)
      controller.abort()
    }
  }, [config, canManage, i18n.language])

  const update = useCallback((path, value) => {
    setConfig((c) => setIn(c, path, value))
    setDirty(true)
    setSavedVersion(null)
  }, [])
  const replace = useCallback((fn) => {
    setConfig((c) => fn(c))
    setDirty(true)
    setSavedVersion(null)
  }, [])

  const issuesBySection = useMemo(() => {
    const counts = {}
    for (const is of issues) {
      const key = is.path.split(/[.[]/)[0]
      counts[key] = (counts[key] || 0) + 1
    }
    return counts
  }, [issues])

  const save = async (e) => {
    e.preventDefault()
    setSaving(true)
    setSaveError(null)
    try {
      const saved = isNew ? await createTenant(slug.trim(), config, note) : await updateTenant(tenant.id, config, note, tenant.current_version)
      setSavedVersion(saved.current_version)
      setDirty(false)
      setNote('')
      if (isNew) {
        navigate(`/tenants/${saved.slug}`, { replace: true, state: { created: true } })
      } else {
        setTenant(saved)
        setConfig(normalizeConfig(saved.config))
      }
    } catch (err) {
      if (err.code === 'CONFIG_INVALID' && err.data?.issues) setIssues(err.data.issues)
      setSaveError(err)
    } finally {
      setSaving(false)
    }
  }

  const doArchive = async () => {
    try {
      await archiveTenant(tenant.id)
      navigate('/tenants', { state: { archived: tenant.name } })
    } catch (err) {
      setSaveError(err)
      setConfirmArchive(false)
    }
  }

  const runPreview = async () => {
    setPreviewing(true)
    setPreviewError(null)
    try {
      setPreview(await previewClaim({ tenantId: tenant?.id, config, claim: toClaim(claim) }))
    } catch (err) {
      setPreview(null)
      setPreviewError(err)
    } finally {
      setPreviewing(false)
    }
  }

  if (loadError) {
    return (
      <div className={ui.page}>
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, loadError, 'tenant')}
        </p>
        <Link to="/tenants">{t('common.back')}</Link>
      </div>
    )
  }
  if (!config || !meta) return <Spinner label={t('common.loading')} />

  const readOnly = !canManage || !!tenant?.archived_at
  const slugIssues = saveError && ['TENANT_SLUG_INVALID', 'TENANT_SLUG_TAKEN'].includes(saveError.code)
  const justCreated = location.state?.created && !dirty && !savedVersion
  const saveDisabled = readOnly || saving || issues.length > 0 || (!isNew && !dirty)
  canSaveRef.current = !saveDisabled
  const jumpTo = (key) => document.getElementById(`section-${key}`)?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })

  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <p className={ui.muted}>
            <Link to="/tenants">← {t('tenants.title')}</Link>
          </p>
          <div className={styles.titleRow}>
            {!isNew && <TenantLogo url={config.branding.logo_url} name={config.branding.company_name || tenant.name} color={config.branding.primary_color} size="md" />}
            <h1 className={ui.title}>{isNew ? t('tenant.newTitle') : config.branding.company_name || tenant.name}</h1>
          </div>
          {!isNew && (
            <p className={ui.subtitle}>
              <span className={ui.mono}>{tenant.slug}</span> · <span className={`${ui.badge} ${ui.badgePrimary}`}>{t('tenant.versionN', { n: tenant.current_version })}</span>{' '}
              {tenant.archived_at && <span className={ui.badge}>{t('tenants.archived')}</span>}
              {dirty && <span className={`${ui.badge} ${ui.badgeWarning}`}>{t('tenant.unsaved')}</span>}
            </p>
          )}
        </div>
        {!isNew && (
          <div className={ui.row}>
            <Link to={`/tenants/${tenant.slug}/history`} className={ui.button}>
              {t('tenants.history')}
            </Link>
            <Link to={`/diff?a=${tenant.slug}`} className={ui.button}>
              {t('tenant.compare')}
            </Link>
            {canManage && !tenant.archived_at && (
              <button type="button" className={`${ui.button} ${ui.danger}`} onClick={() => setConfirmArchive(true)}>
                {t('tenant.archive')}
              </button>
            )}
          </div>
        )}
      </div>

      {justCreated && (
        <p className={ui.success} role="status">
          {t('tenant.created')}
        </p>
      )}
      {savedVersion && (
        <p className={ui.success} role="status">
          {t('tenant.saved', { n: savedVersion })}
        </p>
      )}
      {!canManage && <p className={ui.info}>{t('tenant.readOnly')}</p>}

      <div className={styles.layout}>
        <form ref={formRef} className={styles.form} onSubmit={save} noValidate>
          <nav className={styles.sectionNav} aria-label={t('tenant.sectionNav')}>
            {SECTIONS.map(({ key }) => (
              <button key={key} type="button" className={styles.sectionLink} onClick={() => jumpTo(key)}>
                {t(`sections.${key}`)}
                {issuesBySection[key] > 0 && <span className={styles.issueDot}>{issuesBySection[key]}</span>}
              </button>
            ))}
          </nav>
          <fieldset disabled={readOnly} className={styles.fieldset}>
            {isNew && (
              <section className={ui.card}>
                <div className={ui.field}>
                  <label className={ui.label} htmlFor="tenant-slug">
                    {t('tenant.slug')}
                  </label>
                  <input id="tenant-slug" value={slug} placeholder="carelink" aria-invalid={slugIssues || undefined} onChange={(e) => setSlug(e.target.value.toLowerCase())} />
                  <span className={ui.hint}>{t('tenant.slugHint')}</span>
                </div>
              </section>
            )}

            {SECTIONS.map(({ key, Component }) => (
              <section key={key} className={ui.card} aria-labelledby={`section-${key}`}>
                <h2 id={`section-${key}`} className={`${ui.cardTitle} ${styles.sectionTitle}`}>
                  {t(`sections.${key}`)}
                  {issuesBySection[key] > 0 && <span className={`${ui.badge} ${ui.badgeDanger}`}>{t('tenant.issueCount', { n: issuesBySection[key] })}</span>}
                </h2>
                <Component config={config} update={update} replace={replace} issues={issues} meta={meta} />
              </section>
            ))}
          </fieldset>

          {!readOnly && (
            <div className={`${ui.card} ${styles.saveBar}`}>
              {saveError && (
                <p role="alert" className={ui.alert}>
                  {actionErrorMessage(t, saveError, 'tenant')}{' '}
                  {saveError.code === 'VERSION_CONFLICT' && (
                    <button type="button" className={ui.linkButton} onClick={load}>
                      {t('tenant.reload')}
                    </button>
                  )}
                </p>
              )}
              <div className={styles.saveRow}>
                <input aria-label={t('tenant.note')} placeholder={t('tenant.notePlaceholder')} value={note} onChange={(e) => setNote(e.target.value)} />
                <button type="submit" className={`${ui.button} ${ui.primary}`} disabled={saveDisabled} title={t('tenant.saveShortcut')}>
                  {saving ? t('common.saving') : isNew ? t('tenant.create') : t('tenant.saveVersion')}
                </button>
              </div>
              <p className={`${ui.hint} ${styles.status}`} aria-live="polite">
                <span>{validating ? t('tenant.validating') : issues.length > 0 ? t('tenant.fixIssues', { n: issues.length }) : t('tenant.valid')}</span>
                <kbd className={styles.kbd}>{t('tenant.saveShortcut')}</kbd>
              </p>
              {issues.length > 0 && (
                <details className={styles.issueList}>
                  <summary>{t('tenant.allIssues')}</summary>
                  <ul>
                    {issues.map((is, i) => (
                      <li key={i}>
                        <code>{is.path}</code> — {is.text || is.code}
                      </li>
                    ))}
                  </ul>
                </details>
              )}
            </div>
          )}
        </form>

        <aside className={styles.side}>
          <section className={ui.card} aria-labelledby="preview-title">
            <h2 id="preview-title" className={ui.cardTitle}>
              {t('tenant.previewTitle')}
            </h2>
            <p className={ui.hint}>{t('tenant.previewHint')}</p>
            <ClaimForm value={claim} onChange={setClaim} claimTypes={meta.claim_types} fields={config.custom_fields.filter((f) => f.key)} />
            <div className={styles.previewActions}>
              <button type="button" className={`${ui.button} ${ui.primary}`} onClick={runPreview} disabled={previewing || issues.length > 0}>
                {previewing ? t('common.loading') : t('tenant.runPreview')}
              </button>
            </div>
            {issues.length > 0 && <p className={ui.hint}>{t('tenant.previewNeedsValid')}</p>}
            {previewError && (
              <p role="alert" className={ui.alert}>
                {actionErrorMessage(t, previewError, 'claim')}
              </p>
            )}
          </section>
          {preview && <ResultCard result={preview} />}
        </aside>
      </div>

      <ConfirmDialog
        open={confirmArchive}
        danger
        title={t('tenant.archiveTitle')}
        message={t('tenant.archiveMessage', { name: tenant?.name })}
        confirmLabel={t('tenant.archive')}
        onConfirm={doArchive}
        onCancel={() => setConfirmArchive(false)}
      />
    </div>
  )
}
