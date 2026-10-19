import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import { describeMask, exampleFromMask, maskToRegex, regexToMask } from '../../tenant/pattern'
import TagInput from '../../components/TagInput'
import ui from '../../components/ui.module.css'
import { issuesAt } from '../../tenant/configState'
import styles from './sections.module.css'

/**
 * Format as a mask ("AAA-9999") instead of a regex; the regex is still what
 * gets stored. A stored regex a mask cannot express stays editable as is.
 */
function PatternField({ value, onChange, invalid }) {
  const { t } = useTranslation()
  const mask = regexToMask(value)
  const [raw, setRaw] = useState(mask === null)

  if (raw || mask === null) {
    return (
      <label className={ui.field}>
        <span className={ui.label}>{t('pattern.customLabel')}</span>
        <input value={value} className={ui.mono} aria-invalid={invalid || undefined} onChange={(e) => onChange(e.target.value)} />
        <span className={ui.hint}>
          {t('pattern.customHint')}{' '}
          {regexToMask(value) !== null && (
            <button type="button" className={ui.linkButton} onClick={() => setRaw(false)}>
              {t('pattern.useMask')}
            </button>
          )}
        </span>
      </label>
    )
  }
  return (
    <label className={ui.field}>
      <span className={ui.label}>{t('pattern.label')}</span>
      <input value={mask} placeholder={t('pattern.placeholder')} className={ui.mono} aria-invalid={invalid || undefined} onChange={(e) => onChange(maskToRegex(e.target.value))} />
      <span className={ui.hint}>{mask ? t('pattern.accepts', { example: exampleFromMask(mask), description: describeMask(mask, t) }) : t('pattern.legend')}</span>
      {mask && <span className={ui.hint}>{t('pattern.legend')}</span>}
      <span className={ui.hint}>
        <button type="button" className={ui.linkButton} onClick={() => setRaw(true)}>
          {t('pattern.useRegex')}
        </button>
      </span>
    </label>
  )
}

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
                <input value={f.key} placeholder={t('customFields.keyPlaceholder')} aria-invalid={issuesAt(issues, `${p}.key`).length > 0 || undefined} onChange={(e) => setField(i, { key: e.target.value })} />
                <span className={ui.hint}>{t('customFields.keyHint')}</span>
              </label>
              <label className={ui.field}>
                <span className={ui.label}>{t('customFields.label')}</span>
                <input value={f.label} placeholder={t('customFields.labelPlaceholder')} aria-invalid={issuesAt(issues, `${p}.label`).length > 0 || undefined} onChange={(e) => setField(i, { label: e.target.value })} />
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
              <PatternField value={f.pattern || ''} onChange={(pattern) => setField(i, { pattern })} invalid={issuesAt(issues, `${p}.pattern`).length > 0} />
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
