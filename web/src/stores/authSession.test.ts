import { describe, expect, it } from 'vitest'
import { authSnapshotFromResponse } from './authSession'

describe('auth session helpers', () => {
  it('maps auth API responses to local auth snapshots', () => {
    expect(authSnapshotFromResponse({
      token: 'jwt',
      username: 'alice',
      role: 'admin',
      user_id: 7,
    })).toEqual({
      token: 'jwt',
      username: 'alice',
      role: 'admin',
      userId: 7,
    })
  })
})
