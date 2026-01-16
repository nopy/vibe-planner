import { describe, it, expect, beforeEach } from 'vitest'
import { api } from './api'

describe('API Service', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('exports api instance', () => {
    expect(api).toBeDefined()
    expect(api.defaults).toBeDefined()
  })

  it('has correct baseURL configuration', () => {
    expect(api.defaults.baseURL).toContain('/api')
  })

  it('has correct default headers', () => {
    expect(api.defaults.headers['Content-Type']).toBe('application/json')
  })

  it('has withCredentials enabled', () => {
    expect(api.defaults.withCredentials).toBe(true)
  })
})
