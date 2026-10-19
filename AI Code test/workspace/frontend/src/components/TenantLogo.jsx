import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './TenantLogo.module.css'

/** Only plain web images are rendered; anything else falls back to initials. */
export function isImageUrl(url) {
  return typeof url === 'string' && /^https?:\/\/\S+$/i.test(url.trim())
}

function initials(name) {
  const words = (name || '').trim().split(/\s+/).filter(Boolean)
  return (words.length > 1 ? words[0][0] + words[1][0] : (words[0] || '?').slice(0, 2)).toUpperCase()
}

/**
 * A tenant's logo from branding.logo_url, or its initials on the primary
 * color when the URL is empty or the image cannot be loaded.
 * @param {{ url?: string, name?: string, color?: string, size?: 'sm'|'md'|'lg', onStatus?: (s: 'empty'|'loading'|'ok'|'error') => void }} props
 */
export default function TenantLogo({ url, name, color, size = 'md', onStatus }) {
  const { t } = useTranslation()
  const valid = isImageUrl(url)
  // The load result is remembered per URL, so a new URL starts over without an
  // effect (an effect could run after a cached image's load event and undo it).
  const [loaded, setLoaded] = useState({ url: null, status: null })
  const status = loaded.url === url ? loaded.status : valid ? 'loading' : 'empty'
  const setStatus = (s) => setLoaded({ url, status: s })
  // A cached image may already be complete when React attaches its handlers.
  const checkComplete = (img) => {
    if (img && img.complete && img.naturalWidth > 0 && status === 'loading') setStatus('ok')
  }

  useEffect(() => {
    onStatus?.(status)
  }, [status, onStatus])

  const label = t('branding.logoAlt', { name: name || '' })
  const tryImage = valid && status !== 'error'
  // Initials stay visible until the image has actually loaded, so a slow or
  // broken URL never leaves an empty box.
  return (
    <span className={`${styles.logo} ${styles[size]}`}>
      {tryImage && (
        <img
          key={url}
          ref={checkComplete}
          src={url.trim()}
          alt={label}
          className={`${styles.img} ${status === 'ok' ? '' : styles.pending}`}
          referrerPolicy="no-referrer"
          onLoad={() => setStatus('ok')}
          onError={() => setStatus('error')}
        />
      )}
      {status !== 'ok' && (
        <span
          className={styles.initials}
          style={{ background: /^#[0-9a-f]{6}$/i.test(color || '') ? color : undefined }}
          {...(tryImage ? { 'aria-hidden': true } : { role: 'img', 'aria-label': label })}
        >
          {initials(name)}
        </span>
      )}
    </span>
  )
}
