/**
 * Field formats without regex. People write a mask such as "AAA-9999":
 *   A = capital letter, a = lowercase letter, 9 = digit, * = letter or digit,
 *   anything else is kept as typed.
 * The config still stores a regex (what the backend validates against), so
 * masks are converted on the way in, and regexes are shown as an example
 * ("ABC-1234") with a description on the way out.
 */

const TOKENS = { A: '[A-Z]', a: '[a-z]', 9: '[0-9]', '*': '[A-Za-z0-9]' }
// Regex pieces understood when reading a stored pattern back into a mask.
const CLASS_TO_TOKEN = [
  ['[A-Za-z0-9]', '*'],
  ['[a-zA-Z0-9]', '*'],
  ['[A-Z]', 'A'],
  ['[a-z]', 'a'],
  ['[0-9]', '9'],
  ['\\d', '9'],
]
const META = '.^$*+?()[]{}|\\'

/** Splits "AAA-9999" into runs: [{ch:'A',n:3}, {ch:'-',n:1}, {ch:'9',n:4}] */
function runs(mask) {
  const out = []
  for (const ch of mask) {
    const last = out[out.length - 1]
    if (last && last.ch === ch && TOKENS[ch]) last.n++
    else if (last && !TOKENS[ch] && !TOKENS[last.ch] && last.literal) last.text += ch
    else out.push(TOKENS[ch] ? { ch, n: 1 } : { literal: true, text: ch })
  }
  return out
}

/** "AAA-9999" -> "^[A-Z]{3}-[0-9]{4}$" ('' for an empty mask). */
export function maskToRegex(mask) {
  if (!mask) return ''
  const body = runs(mask)
    .map((r) => (r.literal ? r.text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') : TOKENS[r.ch] + (r.n > 1 ? `{${r.n}}` : '')))
    .join('')
  return `^${body}$`
}

/** "^[A-Z]{3}-[0-9]{4}$" -> "AAA-9999"; null when the regex cannot be written as a mask. */
export function regexToMask(regex) {
  if (!regex) return ''
  if (!regex.startsWith('^') || !regex.endsWith('$') || regex.endsWith('\\$')) return null
  const s = regex.slice(1, -1)
  let out = ''
  let i = 0
  while (i < s.length) {
    let token = null
    for (const [cls, tk] of CLASS_TO_TOKEN) {
      if (s.startsWith(cls, i)) {
        token = tk
        i += cls.length
        break
      }
    }
    let piece
    if (token) piece = token
    else if (s[i] === '\\' && i + 1 < s.length) {
      piece = s[i + 1]
      i += 2
      if (TOKENS[piece]) return null // a literal A/a/9/* has no mask spelling
    } else if (META.includes(s[i])) return null
    else {
      piece = s[i]
      i += 1
      if (TOKENS[piece]) return null
    }
    const q = s.slice(i).match(/^\{(\d{1,2})\}/)
    let n = 1
    if (q) {
      n = Number(q[1])
      i += q[0].length
      if (n < 1) return null
    }
    out += piece.repeat(n)
  }
  return out
}

/** "AAA-9999" -> "ABC-1234" */
export function exampleFromMask(mask) {
  const letters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
  const digits = '1234567890'
  let l = 0
  let d = 0
  let mixed = 0
  return [...(mask || '')]
    .map((ch) => {
      if (ch === 'A') return letters[l++ % 26]
      if (ch === 'a') return letters[l++ % 26].toLowerCase()
      if (ch === '9') return digits[d++ % 10]
      if (ch === '*') return mixed++ % 2 ? digits[d++ % 10] : letters[l++ % 26]
      return ch
    })
    .join('')
}

/** "AAA-9999" -> '3 chữ cái in hoa, "-", 4 chữ số' */
export function describeMask(mask, t) {
  const key = { A: 'pattern.upper', a: 'pattern.lower', 9: 'pattern.digits', '*': 'pattern.alnum' }
  return runs(mask || '')
    .map((r) => (r.literal ? `"${r.text}"` : t(key[r.ch], { n: r.n })))
    .join(', ')
}

/**
 * A stored regex for people: { example: 'ABC-1234', description: '3 chữ cái…' },
 * or { custom: true } when it is a regex a mask cannot express.
 */
export function readablePattern(regex, t) {
  const mask = regexToMask(regex)
  if (mask === null) return { custom: true }
  if (!mask) return { empty: true }
  return { mask, example: exampleFromMask(mask), description: describeMask(mask, t) }
}

/** One line for hints and tables: 'ABC-1234 (3 chữ cái in hoa, "-", 4 chữ số)' */
export function patternText(regex, t) {
  const p = readablePattern(regex, t)
  if (p.empty) return ''
  if (p.custom) return t('pattern.custom')
  return t('pattern.summary', { example: p.example, description: p.description })
}
