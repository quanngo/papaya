import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { diffTenants, getTenant, getVersion, listTenants, listVersions } from '../api/tenants'
import { actionErrorMessage } from '../auth/errorMessages'
import Spinner from '../components/Spinner'
import ui from '../components/ui.module.css'
import ChangesTable, { flatten } from '../tenant/ChangesTable'
import styles from './DiffPage.module.css'

/** "slug@3" -> { tenant: 'slug', version: '3' } */
function parseRef(ref) {
  if (!ref) return { tenant: '', version: '' }
  const i = ref.lastIndexOf('@')
  return i > 0 ? { tenant: ref.slice(0, i), version: ref.slice(i + 1) } : { tenant: ref, version: '' }
}
const toRef = ({ tenant, version }) => (tenant ? (version ? `${tenant}@${version}` : tenant) : '')

function SidePicker({ label, value, onChange, tenants }) {
  const { t } = useTranslation()
  const [versions, setVersions] = useState([])
  useEffect(() => {
    let alive = true
    setVersions([])
    if (value.tenant) listVersions(value.tenant).then((v) => alive && setVersions(v)).catch(() => {})
    return () => {
      alive = false
    }
  }, [value.tenant])

  return (
    <fieldset className={styles.side}>
      <legend className={ui.label}>{label}</legend>
      <select aria-label={t('diff.tenantOf', { side: label })} value={value.tenant} onChange={(e) => onChange({ tenant: e.target.value, version: '' })}>
        <option value="">{t('diff.chooseTenant')}</option>
        {tenants.map((tn) => (
          <option key={tn.id} value={tn.slug}>
            {tn.name}
          </option>
        ))}
      </select>
      <select aria-label={t('diff.versionOf', { side: label })} value={value.version} disabled={!value.tenant} onChange={(e) => onChange({ ...value, version: e.target.value })}>
        <option value="">{t('diff.currentVersion')}</option>
        {versions.map((v) => (
          <option key={v.version} value={String(v.version)}>
            v{v.version} · {t(`history.source.${v.source}`, { n: v.rolled_back_from })}
            {v.is_current ? ` (${t('history.current')})` : ''}
          </option>
        ))}
      </select>
    </fieldset>
  )
}

const loadConfig = async ({ tenant, version }) => (version ? (await getVersion(tenant, version)).config : (await getTenant(tenant)).config)

/** Compare two tenants, or two versions of one tenant, side by side. */
export default function DiffPage() {
  const { t } = useTranslation()
  const [params, setParams] = useSearchParams()
  const a = parseRef(params.get('a'))
  const b = parseRef(params.get('b'))
  const [tenants, setTenants] = useState(null)
  const [diff, setDiff] = useState(null)
  const [configs, setConfigs] = useState(null)
  const [showAll, setShowAll] = useState(false)
  const [error, setError] = useState(null)
  const refA = toRef(a)
  const refB = toRef(b)

  useEffect(() => {
    listTenants(true).then(setTenants).catch(setError)
  }, [])

  useEffect(() => {
    if (!refA || !refB) {
      setDiff(null)
      return undefined
    }
    let alive = true
    setDiff(null)
    setConfigs(null)
    setError(null)
    Promise.all([diffTenants(refA, refB), loadConfig(parseRef(refA)), loadConfig(parseRef(refB))])
      .then(([d, ca, cb]) => {
        if (!alive) return
        setDiff(d)
        setConfigs([ca, cb])
      })
      .catch((err) => alive && setError(err))
    return () => {
      alive = false
    }
  }, [refA, refB])

  const setSide = (key, value) => {
    const next = new URLSearchParams(params)
    const ref = toRef(value)
    if (ref) next.set(key, ref)
    else next.delete(key)
    setParams(next, { replace: true })
  }

  const label = (side) => (side ? `${side.name} · v${side.version}` : '')
  const rows = useMemo(() => {
    if (!configs) return []
    const [l, r] = configs.map((c) => flatten(c))
    const paths = [...new Set([...l.keys(), ...r.keys()])].sort()
    return paths.map((p) => ({ path: p, left: l.get(p) ?? '—', right: r.get(p) ?? '—' }))
  }, [configs])

  if (!tenants && !error) return <Spinner label={t('common.loading')} />
  return (
    <div className={ui.page}>
      <div className={ui.header}>
        <div>
          <h1 className={ui.title}>{t('diff.title')}</h1>
          <p className={ui.subtitle}>{t('diff.subtitle')}</p>
        </div>
      </div>

      <div className={`${ui.card} ${styles.pickers}`}>
        <SidePicker label="A" value={a} onChange={(v) => setSide('a', v)} tenants={tenants || []} />
        <button type="button" className={ui.button} aria-label={t('diff.swap')} onClick={() => setParams(new URLSearchParams({ ...(refB && { a: refB }), ...(refA && { b: refA }) }), { replace: true })} disabled={!refA && !refB}>
          ⇄
        </button>
        <SidePicker label="B" value={b} onChange={(v) => setSide('b', v)} tenants={tenants || []} />
      </div>

      {error && (
        <p role="alert" className={ui.alert}>
          {actionErrorMessage(t, error, 'tenant')}
        </p>
      )}
      {(!refA || !refB) && <p className={`${ui.card} ${ui.empty}`}>{t('diff.pickBoth')}</p>}
      {refA && refB && !diff && !error && <Spinner label={t('common.loading')} />}

      {diff && (
        <section className={ui.card}>
          <div className={ui.header}>
            <h2 className={ui.cardTitle}>{t('diff.count', { n: diff.changes.length })}</h2>
            <label className={ui.row}>
              <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} />
              {t('diff.showAll')}
            </label>
          </div>
          {!showAll ? (
            <ChangesTable changes={diff.changes} leftLabel={label(diff.a)} rightLabel={label(diff.b)} />
          ) : (
            <div className={ui.tableWrap}>
              <table className={ui.table}>
                <thead>
                  <tr>
                    <th>{t('diff.field')}</th>
                    <th>{label(diff.a)}</th>
                    <th>{label(diff.b)}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.path} className={row.left !== row.right ? styles.differs : undefined}>
                      <td className={ui.mono}>{row.path}</td>
                      <td className={ui.mono}>{row.left}</td>
                      <td className={ui.mono}>{row.right}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </div>
  )
}
