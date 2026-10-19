import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './TagInput.module.css'

/**
 * Edits a list of short strings (documents, channels to notify, options):
 * type and press Enter or comma to add, click × to remove.
 */
export default function TagInput({ id, value = [], onChange, placeholder, invalid = false, describedBy, formatLabel }) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState('')
  const list = value || []

  const add = (raw) => {
    const items = raw.split(',').map((s) => s.trim()).filter(Boolean).filter((s) => !list.includes(s))
    if (items.length) onChange([...list, ...items])
    setDraft('')
  }

  return (
    <div className={`${styles.box} ${invalid ? styles.invalid : ''}`}>
      {list.map((item, i) => {
        // Show a readable name; the stored value stays the code (visible on hover).
        const shown = formatLabel ? formatLabel(item) : item
        return (
        <span key={`${item}-${i}`} className={styles.tag} title={shown !== item ? item : undefined}>
          {shown}
          <button type="button" className={styles.remove} aria-label={t('common.removeItem', { item: shown })} onClick={() => onChange(list.filter((_, j) => j !== i))}>
            ×
          </button>
        </span>
        )
      })}
      <input
        id={id}
        className={styles.input}
        value={draft}
        placeholder={list.length ? '' : placeholder}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        onChange={(e) => (e.target.value.includes(',') ? add(e.target.value) : setDraft(e.target.value))}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            add(draft)
          } else if (e.key === 'Backspace' && !draft && list.length) {
            onChange(list.slice(0, -1))
          }
        }}
        onBlur={() => draft && add(draft)}
      />
    </div>
  )
}
