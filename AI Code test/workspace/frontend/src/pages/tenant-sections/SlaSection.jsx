import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import TagInput from '../../components/TagInput'
import ui from '../../components/ui.module.css'
import { issuesAt, toNumber } from '../../tenant/configState'
import styles from './sections.module.css'

const TIMEZONES = ['Asia/Ho_Chi_Minh', 'Asia/Bangkok', 'Asia/Singapore', 'Asia/Jakarta', 'Asia/Manila', 'Asia/Hong_Kong', 'UTC']

/** @param {import('./index').SectionProps} props */
export default function SlaSection({ config, update, replace, issues }) {
  const { t } = useTranslation()
  const s = config.sla

  const setTarget = (ct, raw) =>
    replace((c) => {
      const targets = { ...c.sla.targets }
      if (raw === '') delete targets[ct]
      else targets[ct] = toNumber(raw, 0)
      return { ...c, sla: { ...c.sla, targets } }
    })
  const setEscalation = (list) => update('sla.escalation', list)

  return (
    <div className={ui.stack}>
      <div className={ui.grid2}>
        <div className={ui.field}>
          <label className={ui.label} htmlFor="sla.timezone">
            {t('sla.timezone')}
          </label>
          <input id="sla.timezone" list="sla-timezones" value={s.timezone} aria-invalid={issuesAt(issues, 'sla.timezone').length > 0 || undefined} onChange={(e) => update('sla.timezone', e.target.value)} />
          <datalist id="sla-timezones">
            {TIMEZONES.map((z) => (
              <option key={z} value={z} />
            ))}
          </datalist>
          <FieldIssues issues={issuesAt(issues, 'sla.timezone')} />
        </div>
        <div className={ui.field}>
          <label className={ui.label} htmlFor="sla.default_days">
            {t('sla.defaultDays')}
          </label>
          <input id="sla.default_days" type="number" min="0" value={s.default_days || ''} placeholder="—" onChange={(e) => update('sla.default_days', toNumber(e.target.value, 0))} />
          <span className={ui.hint}>{t('sla.defaultDaysHint')}</span>
          <FieldIssues issues={issuesAt(issues, 'sla.default_days')} />
        </div>
      </div>

      <fieldset className={styles.fieldset}>
        <legend className={ui.label}>{t('sla.targets')}</legend>
        {config.claim_types.enabled.length === 0 && <p className={ui.muted}>{t('sla.noClaimTypes')}</p>}
        <div className={ui.grid2}>
          {config.claim_types.enabled.map((ct) => {
            const path = `sla.targets.${ct}`
            const errs = issuesAt(issues, path)
            return (
              <div key={ct} className={ui.field}>
                <label className={ui.label} htmlFor={path}>
                  {t(`claimTypes.${ct}`)}
                </label>
                <input id={path} type="number" min="1" value={s.targets[ct] ?? ''} placeholder={s.default_days ? String(s.default_days) : ''} aria-invalid={errs.length > 0 || undefined} onChange={(e) => setTarget(ct, e.target.value)} />
                <FieldIssues issues={errs} />
              </div>
            )
          })}
        </div>
      </fieldset>

      <fieldset className={styles.fieldset}>
        <legend className={ui.label}>{t('sla.escalation')}</legend>
        {s.escalation.map((rule, i) => {
          const p = `sla.escalation[${i}]`
          return (
            <div key={i} className={styles.escalationRow}>
              <label className={ui.field}>
                <span className={ui.hint}>{t('sla.afterBreachDays')}</span>
                <input type="number" min="1" value={rule.after_breach_days} onChange={(e) => setEscalation(s.escalation.map((r, j) => (j === i ? { ...r, after_breach_days: toNumber(e.target.value, 0) } : r)))} />
              </label>
              <div className={ui.field}>
                <span className={ui.hint}>{t('sla.notify')}</span>
                <TagInput value={rule.notify} onChange={(v) => setEscalation(s.escalation.map((r, j) => (j === i ? { ...r, notify: v } : r)))} placeholder="team_lead" />
              </div>
              <button type="button" className={`${ui.button} ${ui.small}`} aria-label={t('sla.removeRule', { n: i + 1 })} onClick={() => setEscalation(s.escalation.filter((_, j) => j !== i))}>
                ×
              </button>
              <div className={styles.fullRow}>
                <FieldIssues issues={issuesAt(issues, p, true)} />
              </div>
            </div>
          )
        })}
        <div>
          <button type="button" className={ui.button} onClick={() => setEscalation([...s.escalation, { after_breach_days: 1, notify: [] }])}>
            + {t('sla.addRule')}
          </button>
        </div>
      </fieldset>
    </div>
  )
}
