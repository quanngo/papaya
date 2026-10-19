import ui from './ui.module.css'

/**
 * Validation messages under a field. Texts come from the backend in the
 * user's language (the form re-validates when the language changes).
 */
export default function FieldIssues({ id, issues }) {
  if (!issues?.length) return null
  return (
    <div id={id}>
      {issues.map((i, n) => (
        <p key={`${i.path}-${i.code}-${n}`} className={ui.fieldError} role="alert">
          {i.text || i.code}
        </p>
      ))}
    </div>
  )
}
