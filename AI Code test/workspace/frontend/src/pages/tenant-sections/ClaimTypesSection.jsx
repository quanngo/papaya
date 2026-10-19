import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import TagInput from '../../components/TagInput'
import ui from '../../components/ui.module.css'
import { issuesAt, toggleClaimType } from '../../tenant/configState'
import { docLabel } from '../../tenant/labels'
import styles from './sections.module.css'

/** @param {import('./index').SectionProps} props */
export default function ClaimTypesSection({ config, update, replace, issues, meta }) {
  const { t } = useTranslation()
  const enabled = config.claim_types.enabled

  return (
    <div className={ui.stack}>
      <fieldset className={styles.fieldset}>
        <legend className={ui.label}>{t('claimTypesSection.enabled')}</legend>
        <div className={ui.row}>
          {meta.claim_types.map((ct) => (
            <label key={ct} className={styles.check}>
              <input type="checkbox" checked={enabled.includes(ct)} onChange={(e) => replace((c) => toggleClaimType(c, ct, e.target.checked, meta.claim_types))} />
              {t(`claimTypes.${ct}`)}
            </label>
          ))}
        </div>
        <FieldIssues issues={issuesAt(issues, 'claim_types.enabled', true)} />
      </fieldset>

      {enabled.map((ct) => {
        const base = `claim_types.documents.${ct}`
        const docs = config.claim_types.documents[ct] || { required: [], optional: [] }
        return (
          <div key={ct} className={styles.subcard}>
            <h4 className={styles.subTitle}>{t('claimTypesSection.documentsFor', { type: t(`claimTypes.${ct}`) })}</h4>
            <FieldIssues issues={issuesAt(issues, base)} />
            <div className={ui.grid2}>
              <div className={ui.field}>
                <label className={ui.label} htmlFor={`${base}.required`}>
                  {t('claimTypesSection.required')}
                </label>
                <TagInput id={`${base}.required`} formatLabel={(d) => docLabel(t, d)} value={docs.required} onChange={(v) => update(`${base}.required`, v)} placeholder={t('claimTypesSection.documentPlaceholder')} invalid={issuesAt(issues, `${base}.required`, true).length > 0} />
                <FieldIssues issues={issuesAt(issues, `${base}.required`, true)} />
              </div>
              <div className={ui.field}>
                <label className={ui.label} htmlFor={`${base}.optional`}>
                  {t('claimTypesSection.optional')}
                </label>
                <TagInput id={`${base}.optional`} formatLabel={(d) => docLabel(t, d)} value={docs.optional} onChange={(v) => update(`${base}.optional`, v)} placeholder={t('claimTypesSection.documentPlaceholder')} />
                <FieldIssues issues={issuesAt(issues, `${base}.optional`, true)} />
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
