import { useTranslation } from 'react-i18next'
import ui from '../components/ui.module.css'
import { formatDateTime, formatMoney } from '../i18n/format'
import styles from './ResultCard.module.css'

const routingBadge = { auto_approved: ui.badgeSuccess, manual_review: ui.badgeWarning, rejected: ui.badgeDanger }

/** How one tenant handles one claim (a ProcessResult). */
export default function ResultCard({ result, stored = false }) {
  const { t } = useTranslation()
  const r = result
  const tier = r.approval.tier

  return (
    <article className={`${ui.card} ${styles.card}`} aria-label={r.tenant.name || r.tenant.slug}>
      <header className={styles.head}>
        <div>
          <h3 className={styles.name}>{r.tenant.name || t('preview.unsavedConfig')}</h3>
          <span className={ui.muted}>
            {r.tenant.config_version ? t('tenant.versionN', { n: r.tenant.config_version }) : t('preview.unsavedConfig')}
            {stored && r.claim_id && ` · ${t('preview.stored')}`}
          </span>
        </div>
        <span className={`${ui.badge} ${routingBadge[r.approval.routing]}`}>{t(`routing.${r.approval.routing}`)}</span>
      </header>

      {r.errors.length > 0 && (
        <ul className={styles.errors} role="alert">
          {r.errors.map((e, i) => (
            <li key={i}>{e.text || e.code}</li>
          ))}
        </ul>
      )}

      <dl className={styles.facts}>
        <dt>{t('result.approval')}</dt>
        <dd>
          {r.approval.routing === 'auto_approved' && t('result.autoApproved', { threshold: formatMoney(r.approval.threshold) })}
          {r.approval.routing === 'manual_review' && (
            <>
              <strong>{r.approval.approver_role}</strong>{' '}
              <span className={ui.muted}>
                {t('result.tierRange', { min: formatMoney(tier?.min_amount), max: tier?.max_amount == null ? '∞' : formatMoney(tier.max_amount) })}
              </span>
            </>
          )}
          {r.approval.routing === 'rejected' && '—'}
        </dd>

        <dt>{t('result.notifications')}</dt>
        <dd>
          {r.notifications.length === 0
            ? '—'
            : r.notifications.map((n) => (
                <div key={n.event}>
                  {t(`events.${n.event}`)}: {n.channels.map((c) => t(`channels.${c}`)).join(', ')}
                </div>
              ))}
        </dd>

        <dt>{t('result.documents')}</dt>
        <dd>
          {r.documents.required.length === 0 ? '—' : r.documents.required.join(', ')}
          {r.documents.optional.length > 0 && (
            <div className={ui.muted}>
              {t('result.optional')}: {r.documents.optional.join(', ')}
            </div>
          )}
          {r.documents.missing.length > 0 && (
            <div className={styles.missing}>
              {t('result.missing')}: {r.documents.missing.join(', ')}
            </div>
          )}
        </dd>

        <dt>{t('result.sla')}</dt>
        <dd>
          {r.sla ? (
            <>
              {t('result.slaDays', { days: r.sla.business_days })} → <strong>{formatDateTime(r.sla.deadline, r.sla.timezone)}</strong>
              <div className={ui.muted}>{r.sla.timezone}</div>
            </>
          ) : (
            '—'
          )}
        </dd>

        <dt>{t('result.customFields')}</dt>
        <dd>
          {r.custom_fields.length === 0
            ? t('result.none')
            : r.custom_fields.map((f) => (
                <div key={f.key} className={f.valid ? undefined : styles.missing}>
                  {f.label}
                  {f.required ? '*' : ''}: {f.value === null || f.value === undefined || f.value === '' ? '—' : String(f.value)} {f.valid ? '✓' : `✗ ${f.text || ''}`}
                </div>
              ))}
        </dd>
      </dl>

      {r.trace.length > 0 && (
        <details className={styles.trace}>
          <summary>{t('result.trace')}</summary>
          <ol>
            {r.trace.map((s, i) => (
              <li key={i}>{s.text || s.code}</li>
            ))}
          </ol>
        </details>
      )}
    </article>
  )
}
