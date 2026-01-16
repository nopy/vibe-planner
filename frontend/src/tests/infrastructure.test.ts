import { describe, it, expect } from 'vitest'

describe('Test Infrastructure', () => {
  it('vitest is configured correctly', () => {
    expect(true).toBe(true)
  })

  it('jest-dom matchers are available', () => {
    const element = document.createElement('div')
    element.textContent = 'Hello World'
    document.body.appendChild(element)

    expect(element).toBeInTheDocument()
    expect(element).toHaveTextContent('Hello World')

    document.body.removeChild(element)
  })

  it('localStorage is available and mocked', () => {
    localStorage.setItem('test-key', 'test-value')
    expect(localStorage.getItem('test-key')).toBe('test-value')

    localStorage.removeItem('test-key')
    expect(localStorage.getItem('test-key')).toBeNull()
  })

  it('window.location is mocked', () => {
    expect(window.location).toBeDefined()
    expect(typeof window.location.assign).toBe('function')
    expect(typeof window.location.reload).toBe('function')
  })
})
