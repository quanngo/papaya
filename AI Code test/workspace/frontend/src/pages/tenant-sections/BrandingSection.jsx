import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import TenantLogo from '../../components/TenantLogo'
import ui from '../../components/ui.module.css'
import { issuesAt } from '../../tenant/configState'
import styles from './sections.module.css'

/** @param {import('./index').SectionProps} props */
export default function BrandingSection({ config, update, issues }) {
  const { t } = useTranslation()
  const b = config.branding
  const [logoStatus, setLogoStatus] = useState('empty')
  const colorField = (key) => {
    const path = `branding.${key}`
    const errs = issuesAt(issues, path)
    return (
      <div className={ui.field}>
        <label className={ui.label} htmlFor={path}>
          {t(`branding.${key}`)}
        </label>
        <div className={ui.row}>
          <input type="color" aria-label={t(`branding.${key}`)} value={/^#[0-9a-f]{6}$/i.test(b[key]) ? b[key] : '#000000'} onChange={(e) => update(path, e.target.value)} />
          <input id={path} className={styles.colorText} value={b[key]} aria-invalid={errs.length > 0 || undefined} onChange={(e) => update(path, e.target.value)} />
        </div>
        <FieldIssues issues={errs} />
      </div>
    )
  }

  return (
    <div className={ui.stack}>
      <div className={ui.grid2}>
        <div className={ui.field}>
          <label className={ui.label} htmlFor="branding.company_name">
            {t('branding.company_name')}
          </label>
          <input id="branding.company_name" value={b.company_name} aria-invalid={issuesAt(issues, 'branding.company_name').length > 0 || undefined} onChange={(e) => update('branding.company_name', e.target.value)} />
          <FieldIssues issues={issuesAt(issues, 'branding.company_name')} />
        </div>
        <div className={ui.field}>
          <label className={ui.label} htmlFor="branding.logo_url">
            {t('branding.logo_url')}
          </label>
          <input
            id="branding.logo_url"
            type="url"
            placeholder="https://…"
            value={b.logo_url}
            aria-invalid={issuesAt(issues, 'branding.logo_url').length > 0 || logoStatus === 'error' || undefined}
            aria-describedby="logo-status"
            onChange={(e) => update('branding.logo_url', e.target.value)}
          />
          <FieldIssues issues={issuesAt(issues, 'branding.logo_url')} />
          <div className={styles.logoPreview}>
            <TenantLogo url={b.logo_url} name={b.company_name} color={b.primary_color} size="lg" onStatus={setLogoStatus} />
            <span id="logo-status" className={logoStatus === 'error' ? ui.fieldError : ui.hint} role={logoStatus === 'error' ? 'alert' : undefined}>
              {t(`branding.logoStatus.${logoStatus}`)}
            </span>
          </div>
        </div>
        {colorField('primary_color')}
        {colorField('secondary_color')}
      </div>
      <div className={styles.brandPreview} aria-label={t('branding.preview')}>
        <div className={styles.brandBar} style={{ background: b.primary_color }}>
          {logoStatus === 'ok' ? <img src={b.logo_url.trim()} alt="" className={styles.logo} referrerPolicy="no-referrer" /> : <span className={styles.logoText}>{b.company_name || '—'}</span>}
        </div>
        <div className={styles.brandAccent} style={{ background: b.secondary_color }} />
      </div>
    </div>
  )
}
