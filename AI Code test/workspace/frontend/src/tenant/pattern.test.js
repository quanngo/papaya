import { describe, expect, it } from 'vitest'
import i18n from '../i18n'
import { describeMask, exampleFromMask, maskToRegex, patternText, regexToMask } from './pattern'

const t = i18n.t.bind(i18n)

describe('format masks instead of regex', () => {
  it('converts masks to the stored regex and back', () => {
    for (const [mask, re] of [
      ['AAA-9999', '^[A-Z]{3}-[0-9]{4}$'],
      ['EMP-99999', '^EMP-[0-9]{5}$'],
      ['aa.9', String.raw`^[a-z]{2}\.[0-9]$`],
      ['**-9', '^[A-Za-z0-9]{2}-[0-9]$'],
    ]) {
      expect(maskToRegex(mask)).toBe(re)
      expect(regexToMask(re)).toBe(mask)
    }
    expect(maskToRegex('')).toBe('')
  })

  it('reads the seed regexes as masks', () => {
    expect(regexToMask('^[A-Z]{3}-[0-9]{4}$')).toBe('AAA-9999')
    expect(regexToMask('^EMP-[0-9]{5}$')).toBe('EMP-99999')
    expect(regexToMask(String.raw`^CL-\d{6}$`)).toBe('CL-999999')
  })

  it('gives up on regexes a mask cannot express', () => {
    expect(regexToMask('^(a|b)+$')).toBeNull()
    expect(regexToMask('[0-9]{3}')).toBeNull() // not anchored
    expect(regexToMask(String.raw`^\A$`)).toBeNull() // a literal "A" would read as a letter slot
  })

  it('describes formats in words with an example', () => {
    expect(exampleFromMask('AAA-9999')).toBe('ABC-1234')
    expect(describeMask('AAA-9999', t)).toBe('3 chữ cái in hoa, "-", 4 chữ số')
    expect(patternText('^[A-Z]{3}-[0-9]{4}$', t)).toBe('dạng ABC-1234 (3 chữ cái in hoa, "-", 4 chữ số)')
    expect(patternText('^(a|b)+$', t)).toBe('theo định dạng riêng của công ty')
  })
})
