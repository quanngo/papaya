import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import ui from './ui.module.css'
import styles from './ConfirmDialog.module.css'

/** Modal confirmation. Cancel gets focus first so Enter never confirms by accident. */
export default function ConfirmDialog({ open, title, message, confirmLabel, danger = false, busy = false, onConfirm, onCancel }) {
  const { t } = useTranslation()
  const cancelRef = useRef(null)

  useEffect(() => {
    if (!open) return undefined
    cancelRef.current?.focus()
    const onKey = (e) => e.key === 'Escape' && !busy && onCancel()
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, busy, onCancel])

  if (!open) return null
  return (
    <div className={styles.backdrop} onMouseDown={() => !busy && onCancel()}>
      <div role="dialog" aria-modal="true" aria-labelledby="confirm-title" className={styles.dialog} onMouseDown={(e) => e.stopPropagation()}>
        <h2 id="confirm-title" className={ui.cardTitle}>
          {title}
        </h2>
        <p className={styles.message}>{message}</p>
        <div className={styles.actions}>
          <button ref={cancelRef} type="button" className={ui.button} onClick={onCancel} disabled={busy}>
            {t('common.cancel')}
          </button>
          <button type="button" className={`${ui.button} ${danger ? ui.danger : ui.primary}`} onClick={onConfirm} disabled={busy}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
