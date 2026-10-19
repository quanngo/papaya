import { useTranslation } from 'react-i18next'
import TagInput from '../components/TagInput'
import ui from '../components/ui.module.css'
import { toNumber } from './configState'
import { docLabel } from './labels'
import { patternText, readablePattern } from './pattern'

/**
 * @typedef {Object} ClaimDraft
 * @property {string} claim_type
 * @property {string|number} amount
 * @property {string} submitted_at   datetime-local value ('' = now)
 * @property {string[]} documents
 * @property {Object<string, string>} custom_fields
 */

export const emptyClaim = () => ({ claim_type: 'OUTPATIENT', amount: 15000, submitted_at: '', documents: ['invoice', 'prescription'], custom_fields: {} })

/** Draft -> API claim (custom fields left empty are omitted). */
export function toClaim(draft) {
  const custom = Object.fromEntries(Object.entries(draft.custom_fields || {}).filter(([, v]) => v !== '' && v !== undefined))
  return {
    claim_type: draft.claim_type,
    amount: toNumber(draft.amount, 0),
    submitted_at: draft.submitted_at ? new Date(draft.submitted_at).toISOString() : undefined,
    documents: draft.documents,
    custom_fields: custom,
  }
}

/**
 * Sample-claim form. `fields` are the custom fields to ask for (the union of
 * the selected tenants' fields), so a claim can be identical across tenants.
 */
export default function ClaimForm({ value, onChange, claimTypes, fields = [] }) {
  const { t } = useTranslation()
  const set = (key, v) => onChange({ ...value, [key]: v })

  return (
    <div className={ui.stack}>
      <div className={ui.grid2}>
        <label className={ui.field}>
          <span className={ui.label}>{t('claim.claimType')}</span>
          <select value={value.claim_type} onChange={(e) => set('claim_type', e.target.value)}>
            {claimTypes.map((ct) => (
              <option key={ct} value={ct}>
                {t(`claimTypes.${ct}`)}
              </option>
            ))}
          </select>
        </label>
        <label className={ui.field}>
          <span className={ui.label}>{t('claim.amount')}</span>
          <input type="number" min="0" step="any" inputMode="decimal" value={value.amount} onChange={(e) => set('amount', e.target.value)} />
        </label>
        <label className={ui.field}>
          <span className={ui.label}>{t('claim.submittedAt')}</span>
          <input type="datetime-local" value={value.submitted_at} onChange={(e) => set('submitted_at', e.target.value)} />
          <span className={ui.hint}>{t('claim.submittedAtHint')}</span>
        </label>
      </div>
      <div className={ui.field}>
        <label className={ui.label} htmlFor="claim-documents">
          {t('claim.documents')}
        </label>
        <TagInput id="claim-documents" formatLabel={(d) => docLabel(t, d)} value={value.documents} onChange={(v) => set('documents', v)} placeholder={t('claim.documentsPlaceholder')} />
      </div>
      {fields.length > 0 && (
        <div className={ui.grid2}>
          {fields.map((f) => (
            <label key={f.key} className={ui.field}>
              <span className={ui.label}>
                {f.label} {f.required && <span className={ui.muted}>*</span>}
              </span>
              {f.type === 'select' ? (
                <select value={value.custom_fields[f.key] ?? ''} onChange={(e) => set('custom_fields', { ...value.custom_fields, [f.key]: e.target.value })}>
                  <option value="">—</option>
                  {(f.options || []).map((o) => (
                    <option key={o} value={o}>
                      {o}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  type={f.type === 'number' ? 'number' : 'text'}
                  value={value.custom_fields[f.key] ?? ''}
                  placeholder={readablePattern(f.pattern, t).example ? t('claim.fieldExample', { example: readablePattern(f.pattern, t).example }) : t('claim.fieldPlaceholder', { label: f.label })}
                  onChange={(e) => set('custom_fields', { ...value.custom_fields, [f.key]: e.target.value })}
                />
              )}
              {f.pattern && <span className={ui.hint}>{t('claim.patternHint', { pattern: patternText(f.pattern, t) })}</span>}
              {f.tenants && <span className={ui.hint}>{t('claim.fieldFor', { tenants: f.tenants.join(', ') })}</span>}
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
