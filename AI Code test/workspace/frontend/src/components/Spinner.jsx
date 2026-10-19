import styles from './Spinner.module.css'

/** Loading indicator; `label` is read by screen readers. */
export default function Spinner({ label, inline = false }) {
  return (
    <div role="status" className={inline ? styles.inline : styles.block}>
      <span className={styles.dot} aria-hidden="true" />
      <span className={inline ? styles.text : 'visually-hidden'}>{label}</span>
    </div>
  )
}
