import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import ui from '../../components/ui.module.css'
import { chainTiers, issuesAt, toNumber } from '../../tenant/configState'
import styles from './sections.module.css'

/**
 * Threshold + tiers. Tier minimums are derived (first = threshold, then the
 * previous maximum) so tiers stay contiguous; the last tier has no maximum.
 * @param {import('./index').SectionProps} props
 */
export default function ApprovalSection({ config, replace, issues }) {
  const { t } = useTranslation()
  const a = config.approval
  const setApproval = (next) => replace((c) => ({ ...c, approval: chainTiers(next) }))
  const setTier = (i, patch) => setApproval({ ...a, tiers: a.tiers.map((tier, j) => (j === i ? { ...tier, ...patch } : tier)) })
  const last = a.tiers.length - 1

  // The former last tier now needs a maximum; validation points at it until it is filled.
  const addTier = () => setApproval({ ...a, tiers: [...a.tiers, { min_amount: 0, max_amount: null, role: '' }] })
  const removeTier = (i) => {
    const tiers = a.tiers.filter((_, j) => j !== i)
    if (tiers.length) tiers[tiers.length - 1] = { ...tiers[tiers.length - 1], max_amount: null }
    setApproval({ ...a, tiers })
  }

  const thresholdIssues = issuesAt(issues, 'approval.auto_approval_threshold')
  return (
    <div className={ui.stack}>
      <div className={ui.field}>
        <label className={ui.label} htmlFor="approval.auto_approval_threshold">
          {t('approval.threshold')}
        </label>
        <input
          id="approval.auto_approval_threshold"
          type="number"
          step="any"
          className={styles.narrow}
          value={a.auto_approval_threshold}
          aria-invalid={thresholdIssues.length > 0 || undefined}
          onChange={(e) => setApproval({ ...a, auto_approval_threshold: toNumber(e.target.value, 0) })}
        />
        <span className={ui.hint}>{t('approval.thresholdHint')}</span>
        <FieldIssues issues={thresholdIssues} />
      </div>

      <div className={ui.tableWrap}>
        <table className={ui.table}>
          <thead>
            <tr>
              <th>#</th>
              <th>{t('approval.from')}</th>
              <th>{t('approval.to')}</th>
              <th>{t('approval.role')}</th>
              <th>
                <span className="visually-hidden">{t('common.actions')}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {a.tiers.map((tier, i) => {
              const p = `approval.tiers[${i}]`
              const rowIssues = issuesAt(issues, p, true)
              return (
                <tr key={i}>
                  <td>{i + 1}</td>
                  <td className={styles.numCell}>
                    <input aria-label={t('approval.fromN', { n: i + 1 })} value={tier.min_amount} readOnly tabIndex={-1} className={styles.readonly} />
                  </td>
                  <td className={styles.numCell}>
                    {i === last ? (
                      <input aria-label={t('approval.toN', { n: i + 1 })} value="∞" readOnly tabIndex={-1} className={styles.readonly} />
                    ) : (
                      <input
                        aria-label={t('approval.toN', { n: i + 1 })}
                        type="number"
                        step="any"
                        value={tier.max_amount ?? ''}
                        aria-invalid={issuesAt(issues, `${p}.max_amount`).length > 0 || undefined}
                        onChange={(e) => setTier(i, { max_amount: toNumber(e.target.value, null) })}
                      />
                    )}
                  </td>
                  <td>
                    <input
                      aria-label={t('approval.roleN', { n: i + 1 })}
                      value={tier.role}
                      placeholder="assessor"
                      aria-invalid={issuesAt(issues, `${p}.role`).length > 0 || undefined}
                      onChange={(e) => setTier(i, { role: e.target.value })}
                    />
                    <FieldIssues issues={rowIssues} />
                  </td>
                  <td>
                    <button type="button" className={`${ui.button} ${ui.small}`} onClick={() => removeTier(i)} aria-label={t('approval.removeTier', { n: i + 1 })}>
                      ×
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <FieldIssues issues={issuesAt(issues, 'approval.tiers')} />
      <div>
        <button type="button" className={ui.button} onClick={addTier}>
          + {t('approval.addTier')}
        </button>
      </div>
    </div>
  )
}
