import { useTranslation } from 'react-i18next'
import ui from '../components/ui.module.css'
import styles from './ChangesTable.module.css'

export function formatValue(v) {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

/** Differences from GET /tenants/diff, grouped by config module. */
export default function ChangesTable({ changes, leftLabel, rightLabel }) {
  const { t } = useTranslation()
  if (!changes.length) return <p className={ui.success}>{t('diff.identical')}</p>

  const groups = []
  for (const c of changes) {
    const g = groups.find((x) => x.module === c.module)
    if (g) g.items.push(c)
    else groups.push({ module: c.module, items: [c] })
  }

  return (
    <div className={ui.tableWrap}>
      <table className={ui.table}>
        <thead>
          <tr>
            <th>{t('diff.field')}</th>
            <th>{leftLabel}</th>
            <th>{rightLabel}</th>
          </tr>
        </thead>
        {groups.map((g) => (
          <tbody key={g.module}>
            <tr>
              <th colSpan={3} className={styles.group}>
                {t(`sections.${g.module}`, { defaultValue: g.module })} <span className={ui.badge}>{g.items.length}</span>
              </th>
            </tr>
            {g.items.map((c, i) => (
              <tr key={i} className={styles[c.kind]}>
                <td>
                  <code className={styles.path}>{c.path}</code>
                  <span className={styles.kind}>{t(`diff.kind.${c.kind}`)}</span>
                </td>
                <td className={ui.mono}>{formatValue(c.left)}</td>
                <td className={ui.mono}>{formatValue(c.right)}</td>
              </tr>
            ))}
          </tbody>
        ))}
      </table>
    </div>
  )
}

/** Flattens a config to [path, value] rows for the full side-by-side view. */
export function flatten(value, prefix = '', out = new Map()) {
  if (Array.isArray(value)) {
    if (value.length === 0) out.set(prefix, '[]')
    value.forEach((v, i) => flatten(v, `${prefix}[${i}]`, out))
  } else if (value && typeof value === 'object') {
    const keys = Object.keys(value).sort()
    if (keys.length === 0) out.set(prefix, '{}')
    keys.forEach((k) => flatten(value[k], prefix ? `${prefix}.${k}` : k, out))
  } else {
    out.set(prefix, value === null || value === undefined || value === '' ? '—' : String(value))
  }
  return out
}
