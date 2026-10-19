import { describe, expect, it } from 'vitest'
import vi from './vi.json'
import en from './en.json'

const keys = (obj, prefix = '') =>
  Object.entries(obj).flatMap(([k, v]) => (v && typeof v === 'object' ? keys(v, `${prefix}${k}.`) : [`${prefix}${k}`]))

describe('locale catalogs', () => {
  it('vi and en have exactly the same keys', () => {
    expect(keys(en).sort()).toEqual(keys(vi).sort())
  })

  it('every backend error code the UI can show is translated', () => {
    for (const code of ['UNAUTHENTICATED', 'FORBIDDEN', 'CROSS_ORIGIN', 'NETWORK_ERROR', 'TENANT_NOT_FOUND', 'CONFIG_INVALID', 'INTERNAL_ERROR']) {
      expect(vi.errors.common[code], code).toBeTruthy()
    }
    for (const code of ['VERSION_CONFLICT', 'TENANT_SLUG_TAKEN', 'TENANT_SLUG_INVALID']) expect(vi.errors.tenant[code], code).toBeTruthy()
  })
})
