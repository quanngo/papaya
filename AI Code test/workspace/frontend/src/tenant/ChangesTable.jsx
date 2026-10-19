import { useTranslation } from 'react-i18next'
import ui from '../components/ui.module.css'
import { describePath, describeValue } from './humanize'
import styles from './ChangesTable.module.css'

const kindBadge = { changed: ui.badgeWarning, added: ui.badgeSuccess, removed: ui.badgeDanger }

/** Groups rows by config area (Branding, Approval rules, …) keeping API order. */
export function groupBySection(rows, t) {
  const groups = []
  for (const row of rows) {
    const { section, label } = describePath(row.path, t)
    let g = groups.find((x) => x.section === section)
    if (!g) groups.push((g = { section, items: [] }))
    g.items.push({ ...row, label })
  }
  return groups
}

/** Differences from GET /tenants/diff, in plain words, grouped by config area. */
export default function ChangesTable({ changes, leftLabel, rightLabel }) {
  const { t } = useTranslation()
  if (!changes.length) return <p className={ui.success}>{t('diff.identical')}</p>
  // The empty side of an added/removed row is "not set", except an empty tier
  // maximum, which means "no limit".
  const side = (c, emptyKind, value) =>
    c.kind === emptyKind && !c.path.endsWith('max_amount') ? <span className={ui.muted}>{t('diff.value.notSet')}</span> : describeValue(c.path, value ?? null, t)

  return (
    <div className={ui.tableWrap}>
      <table className={`${ui.table} ${styles.table}`}>
        <thead>
          <tr>
            <th>{t('diff.field')}</th>
            <th>{leftLabel}</th>
            <th>{rightLabel}</th>
          </tr>
        </thead>
        {groupBySection(changes, t).map((g) => (
          <tbody key={g.section}>
            <tr>
              <th colSpan={3} className={styles.group}>
                {g.section} <span className={ui.badge}>{g.items.length}</span>
              </th>
            </tr>
            {g.items.map((c, i) => (
              <tr key={i} className={styles[c.kind]} title={c.path}>
                <td>
                  <span className={styles.label}>{c.label || g.section}</span>
                  <span className={`${ui.badge} ${kindBadge[c.kind]} ${styles.kind}`}>{t(`diff.kind.${c.kind}`)}</span>
                </td>
                <td>{side(c, 'added', c.left)}</td>
                <td>{side(c, 'removed', c.right)}</td>
              </tr>
            ))}
          </tbody>
        ))}
      </table>
    </div>
  )
}
