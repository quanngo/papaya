import { useEffect, useMemo, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { getTenant, listTenants } from '../api/tenants'
import { fetchMeta } from '../api/configs'
import { previewClaim, processBatch } from '../api/claims'
import { useAuth } from '../auth/AuthContext'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import ClaimForm, { emptyClaim, toClaim } from '../tenant/ClaimForm'
import ResultCard from '../tenant/ResultCard'
import styles from './PreviewPage.module.css'

/**
 * One sample claim against one or more tenants: "Preview" predicts the
 * outcome (nothing stored); "Process" runs processClaim and stores the claims.
 */
export default function PreviewPage() {
  const { t, i18n } = useTranslation()
  const { can } = useAuth()
  const [tenants, setTenants] = useState(null)
  const [meta, setMeta] = useState(null)
  const [configs, setConfigs] = useState({})
  const [selected, setSelected] = useState([])
  const [claim, setClaim] = useState(emptyClaim())
  const [results, setResults] = useState(null)
  const [stored, setStored] = useState(false)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const lastPreview = useRef(null)
  const [params] = useSearchParams()
  const preselect = params.get('tenant')

  useEffect(() => {
    Promise.all([listTenants(), fetchMeta()])
      .then(([list, m]) => {
        setTenants(list)
        setMeta(m)
        // ?tenant=<slug|id> (from the insurer list) preselects one insurer; otherwise all.
        const only = list.find((tn) => tn.slug === preselect || tn.id === preselect)
        setSelected(only ? [only.id] : list.map((tn) => tn.id))
      })
      .catch(setError)
  }, [])

  // Load the configs of selected tenants to know which custom fields to ask for.
  useEffect(() => {
    const missing = selected.filter((id) => !configs[id])
    if (!missing.length) return
    Promise.all(missing.map((id) => getTenant(id)))
      .then((list) => setConfigs((c) => ({ ...c, ...Object.fromEntries(list.map((d) => [d.id, d])) })))
      .catch(setError)
  }, [selected, configs])

  const fields = useMemo(() => {
    const byKey = new Map()
    for (const id of selected) {
      const d = configs[id]
      for (const f of d?.config.custom_fields || []) {
        const prev = byKey.get(f.key)
        byKey.set(f.key, prev ? { ...prev, required: prev.required || f.required, tenants: [...prev.tenants, d.name] } : { ...f, tenants: [d.name] })
      }
    }
    return [...byKey.values()]
  }, [selected, configs])

  const runPreview = async (ids = selected, draft = claim) => {
    setBusy(true)
    setError(null)
    try {
      const c = toClaim(draft)
      setResults(await Promise.all(ids.map((id) => previewClaim({ tenantId: id, claim: c }))))
      setStored(false)
      lastPreview.current = { ids, draft }
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  const runProcess = async () => {
    setBusy(true)
    setError(null)
    try {
      setResults(await processBatch(selected, toClaim(claim)))
      setStored(true)
      lastPreview.current = null
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  // Preview texts come from the backend: re-run the last preview in the new language.
  useEffect(() => {
    if (lastPreview.current) runPreview(lastPreview.current.ids, lastPreview.current.draft)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [i18n.language])

  if (!tenants || !meta) {
    return error ? (
      <div className={ui.page}>
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error)}
        </p>
      </div>
    ) : (
      <Spinner label={t('common.loading')} />
    )
  }

  const toggle = (id, on) => setSelected((s) => (on ? tenants.map((tn) => tn.id).filter((x) => x === id || s.includes(x)) : s.filter((x) => x !== id)))

  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <h1 className={ui.title}>{t('preview.title')}</h1>
          <p className={ui.subtitle}>{t('preview.subtitle')}</p>
        </div>
      </div>

      <section className={ui.card}>
        <fieldset className={styles.tenants}>
          <legend className={ui.label}>{t('preview.tenants')}</legend>
          <div className={ui.row}>
            {tenants.map((tn) => (
              <label key={tn.id} className={styles.check}>
                <input type="checkbox" checked={selected.includes(tn.id)} onChange={(e) => toggle(tn.id, e.target.checked)} />
                {tn.name}
              </label>
            ))}
          </div>
        </fieldset>
        <ClaimForm value={claim} onChange={setClaim} claimTypes={meta.claim_types} fields={fields} />
        <div className={styles.actions}>
          <button type="button" className={`${ui.button} ${ui.primary}`} disabled={busy || selected.length === 0} onClick={() => runPreview()}>
            {t('preview.run', { n: selected.length })}
          </button>
          {can('claims.process') && (
            <button type="button" className={ui.button} disabled={busy || selected.length === 0} onClick={runProcess}>
              {t('preview.process', { n: selected.length })}
            </button>
          )}
          {busy && <Spinner inline label={t('common.loading')} />}
        </div>
        <p className={ui.hint}>{t('preview.processHint')}</p>
      </section>

      {error && (
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error, 'claim')}
        </p>
      )}
      {results && (
        <>
          <h2 className={styles.resultsTitle}>{stored ? t('preview.processedTitle') : t('preview.resultsTitle')}</h2>
          <div className={styles.results}>
            {results.map((r) => (
              <ResultCard key={r.tenant.id} result={r} stored={stored} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
