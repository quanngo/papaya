import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import TagInput from '../../components/TagInput'
import ui from '../../components/ui.module.css'
import { issuesAt } from '../../tenant/configState'
import styles from './sections.module.css'

/** @param {import('./index').SectionProps} props */
export default function CustomFieldsSection({ config, update, issues, meta }) {
  const { t } = useTranslation()
  const fields = config.custom_fields
  const setField = (i, patch) => update('custom_fields', fields.map((f, j) => (j === i ? { ...f, ...patch } : f)))

  return (
    <div className={ui.stack}>
      {fields.length === 0 && <p className={ui.muted}>{t('customFields.none')}</p>}
      {fields.map((f, i) => {
        const p = `custom_fields[${i}]`
        return (
          <div key={i} className={styles.subcard}>
            <div className={styles.fieldGrid}>
              <label className={ui.field}>
                <span className={ui.label}>{t('customFields.key')}</span>
                <input value={f.key} placeholder="employee_id" aria-invalid={issuesAt(issues, `${p}.key`).length > 0 || undefined} onChange={(e) => setField(i, { key: e.target.value })} />
              </label>
              <label className={ui.field}>
                <span className={ui.label}>{t('customFields.label')}</span>
                <input value={f.label} placeholder="Employee ID" aria-invalid={issuesAt(issues, `${p}.label`).length > 0 || undefined} onChange={(e) => setField(i, { label: e.target.value })} />
              </label>
              <label className={ui.field}>
                <span className={ui.label}>{t('customFields.type')}</span>
                <select value={f.type} onChange={(e) => setField(i, { type: e.target.value })}>
                  {meta.custom_field_types.map((ty) => (
                    <option key={ty} value={ty}>
                      {t(`fieldTypes.${ty}`)}
                    </option>
                  ))}
                </select>
              </label>
              <label className={ui.field}>
                <span className={ui.label}>{t('customFields.pattern')}</span>
                <input value={f.pattern || ''} placeholder="^[A-Z]{3}-[0-9]{4}$" className={ui.mono} onChange={(e) => setField(i, { pattern: e.target.value })} />
              </label>
            </div>
            {f.type === 'select' && (
              <div className={ui.field}>
                <span className={ui.label}>{t('customFields.options')}</span>
                <TagInput value={f.options || []} onChange={(v) => setField(i, { options: v })} placeholder={t('customFields.optionsPlaceholder')} />
              </div>
            )}
            <div className={styles.fieldFooter}>
              <label className={styles.check}>
                <input type="checkbox" checked={f.required} onChange={(e) => setField(i, { required: e.target.checked })} />
                {t('customFields.required')}
              </label>
              <button type="button" className={`${ui.button} ${ui.small} ${ui.danger}`} onClick={() => update('custom_fields', fields.filter((_, j) => j !== i))}>
                {t('customFields.remove')}
              </button>
            </div>
            <FieldIssues issues={issuesAt(issues, p, true)} />
          </div>
        )
      })}
      <div>
        <button type="button" className={ui.button} onClick={() => update('custom_fields', [...fields, { key: '', label: '', type: 'text', required: true }])}>
          + {t('customFields.add')}
        </button>
      </div>
    </div>
  )
}
