import { describe, expect, it } from 'vitest'
import { chainTiers, issuesAt, parsePath, setIn, toggleClaimType } from './configState'

describe('configState', () => {
  it('parses backend issue paths', () => {
    expect(parsePath('approval.tiers[1].min_amount')).toEqual(['approval', 'tiers', 1, 'min_amount'])
    expect(parsePath('sla.targets.DENTAL')).toEqual(['sla', 'targets', 'DENTAL'])
  })

  it('sets values immutably', () => {
    const before = { approval: { tiers: [{ role: 'a' }, { role: 'b' }] } }
    const after = setIn(before, 'approval.tiers[1].role', 'lead')
    expect(after.approval.tiers[1].role).toBe('lead')
    expect(before.approval.tiers[1].role).toBe('b')
    expect(after.approval.tiers[0]).toBe(before.approval.tiers[0])
  })

  it('keeps tiers contiguous from the threshold', () => {
    const a = chainTiers({
      auto_approval_threshold: 1000,
      tiers: [
        { min_amount: 0, max_amount: 5000, role: 'a' },
        { min_amount: 0, max_amount: null, role: 'b' },
      ],
    })
    expect(a.tiers.map((t) => t.min_amount)).toEqual([1000, 5000])
  })

  it('matches issues by exact path or nested path', () => {
    const issues = [{ path: 'approval.tiers[1].min_amount' }, { path: 'approval.tiers' }]
    expect(issuesAt(issues, 'approval.tiers')).toHaveLength(1)
    expect(issuesAt(issues, 'approval.tiers', true)).toHaveLength(2)
    expect(issuesAt(issues, 'approval.tiers[1]', true)).toHaveLength(1)
  })

  it('enables a claim type in meta order with a documents entry', () => {
    const c = toggleClaimType({ claim_types: { enabled: ['INPATIENT'], documents: {} } }, 'OUTPATIENT', true, ['OUTPATIENT', 'INPATIENT'])
    expect(c.claim_types.enabled).toEqual(['OUTPATIENT', 'INPATIENT'])
    expect(c.claim_types.documents.OUTPATIENT.required).toEqual(['invoice'])
  })
})
