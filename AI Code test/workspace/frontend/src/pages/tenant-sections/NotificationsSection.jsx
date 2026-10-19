import { useTranslation } from 'react-i18next'
import FieldIssues from '../../components/FieldIssues'
import ui from '../../components/ui.module.css'
import { issuesAt } from '../../tenant/configState'
import styles from './sections.module.css'

/** @param {import('./index').SectionProps} props */
export default function NotificationsSection({ config, update, issues, meta }) {
  const { t } = useTranslation()
  const n = config.notifications
  const rule = (e) => n.events[e] || { enabled: false, channels: [], template: '' }

  const toggleChannel = (e, ch, on) => {
    const r = rule(e)
    const channels = on ? meta.channels.filter((c) => c === ch || r.channels.includes(c)) : r.channels.filter((c) => c !== ch)
    update(`notifications.events.${e}`, { ...r, channels })
  }

  return (
    <div className={ui.stack}>
      <div className={ui.tableWrap}>
        <table className={ui.table}>
          <thead>
            <tr>
              <th>{t('notifications.event')}</th>
              <th>{t('notifications.enabled')}</th>
              {meta.channels.map((ch) => (
                <th key={ch}>{t(`channels.${ch}`)}</th>
              ))}
              <th>{t('notifications.template')}</th>
            </tr>
          </thead>
          <tbody>
            {meta.notification_events.map((e) => {
              const r = rule(e)
              const base = `notifications.events.${e}`
              return (
                <tr key={e}>
                  <td>
                    {t(`events.${e}`)}
                    <FieldIssues issues={issuesAt(issues, base, true)} />
                  </td>
                  <td>
                    <input type="checkbox" aria-label={t('notifications.enableEvent', { event: t(`events.${e}`) })} checked={r.enabled} onChange={(ev) => update(base, { ...r, enabled: ev.target.checked })} />
                  </td>
                  {meta.channels.map((ch) => (
                    <td key={ch}>
                      <input
                        type="checkbox"
                        aria-label={t('notifications.channelFor', { channel: t(`channels.${ch}`), event: t(`events.${e}`) })}
                        checked={r.channels.includes(ch)}
                        disabled={!r.enabled}
                        onChange={(ev) => toggleChannel(e, ch, ev.target.checked)}
                      />
                    </td>
                  ))}
                  <td className={styles.templateCell}>
                    <input
                      aria-label={t('notifications.templateFor', { event: t(`events.${e}`) })}
                      value={r.template || ''}
                      placeholder={t('notifications.defaultTemplate')}
                      disabled={!r.enabled}
                      onChange={(ev) => update(base, { ...r, template: ev.target.value })}
                    />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <div className={ui.field}>
        <label className={ui.label} htmlFor="notifications.webhook_url">
          {t('notifications.webhookUrl')}
        </label>
        <input
          id="notifications.webhook_url"
          type="url"
          placeholder="https://…"
          value={n.webhook_url || ''}
          aria-invalid={issuesAt(issues, 'notifications.webhook_url').length > 0 || undefined}
          onChange={(e) => update('notifications.webhook_url', e.target.value)}
        />
        <span className={ui.hint}>{t('notifications.webhookHint')}</span>
        <FieldIssues issues={issuesAt(issues, 'notifications.webhook_url')} />
      </div>
    </div>
  )
}
