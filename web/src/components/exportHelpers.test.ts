import { describe, expect, it } from 'vitest'
import { exportIDsForChoice, selectedExportIDs } from './exportHelpers'

describe('export dialog helpers', () => {
  it('keeps only named monitors for selected exports', () => {
    expect(selectedExportIDs([
      { id: 1, name: 'site' },
      { id: 2, name: '' },
      { id: 3, name: 'api' },
    ])).toEqual([1, 3])
  })

  it('derives export ids from the chosen range', () => {
    const monitors = [{ id: 1, name: 'site' }]

    expect(exportIDsForChoice('all', monitors)).toBeUndefined()
    expect(exportIDsForChoice('selected', monitors)).toEqual([1])
  })
})
