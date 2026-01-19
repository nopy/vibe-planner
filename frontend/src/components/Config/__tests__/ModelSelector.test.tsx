import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ModelSelector } from '../ModelSelector'

describe('ModelSelector', () => {
  const mockOnChange = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders with initial provider and model', () => {
    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const selects = screen.getAllByRole('combobox')
    const providerSelect = selects[0]
    const modelSelect = selects[1]

    expect(providerSelect).toHaveValue('openai')
    expect(modelSelect).toHaveValue('gpt-4o-mini')
  })

  it('changing provider auto-selects default model for that provider', async () => {
    const user = userEvent.setup()

    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const providerSelect = screen.getAllByRole('combobox')[0]

    await user.selectOptions(providerSelect, 'anthropic')

    expect(mockOnChange).toHaveBeenCalledWith('anthropic', 'claude-3-5-sonnet-20240620')
  })

  it('changing model calls onChange with correct values', async () => {
    const user = userEvent.setup()

    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const modelSelect = screen.getAllByRole('combobox')[1]

    await user.selectOptions(modelSelect, 'gpt-4o')

    expect(mockOnChange).toHaveBeenCalledWith('openai', 'gpt-4o')
  })

  it('custom provider shows text input instead of dropdown', () => {
    render(
      <ModelSelector
        value={{ provider: 'custom', name: 'llama-3-70b' }}
        onChange={mockOnChange}
      />
    )

    const modelInput = screen.getByPlaceholderText(/e\.g\. llama-3-70b/i)

    expect(modelInput).toBeInTheDocument()
    expect(modelInput).toHaveValue('llama-3-70b')
    expect(modelInput.tagName).toBe('INPUT')
  })

  it('custom provider text input updates model name', async () => {
    const user = userEvent.setup()

    render(
      <ModelSelector
        value={{ provider: 'custom', name: '' }}
        onChange={mockOnChange}
      />
    )

    const modelInput = screen.getByPlaceholderText(/e\.g\. llama-3-70b/i)

    await user.type(modelInput, 'mistral-7b')

    expect(mockOnChange).toHaveBeenLastCalledWith('custom', 'mistral-7b')
  })

  it('disabled prop disables all inputs', () => {
    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
        disabled={true}
      />
    )

    const selects = screen.getAllByRole('combobox')
    const providerSelect = selects[0]
    const modelSelect = selects[1]

    expect(providerSelect).toBeDisabled()
    expect(modelSelect).toBeDisabled()
  })

  it('provider dropdown has correct options', () => {
    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const providerSelect = screen.getAllByRole('combobox')[0]
    const options = Array.from(providerSelect.querySelectorAll('option')).map(
      (opt) => opt.value
    )

    expect(options).toEqual(['openai', 'anthropic', 'custom'])
  })

  it('OpenAI models list rendered correctly', () => {
    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const modelSelect = screen.getAllByRole('combobox')[1]
    const options = Array.from(modelSelect.querySelectorAll('option')).map(
      (opt) => opt.value
    )

    expect(options).toContain('gpt-4o-mini')
    expect(options).toContain('gpt-4o')
    expect(options).toContain('gpt-4')
    expect(options).toContain('gpt-3.5-turbo')
  })

  it('Anthropic models list rendered correctly', () => {
    render(
      <ModelSelector
        value={{ provider: 'anthropic', name: 'claude-3-5-sonnet-20240620' }}
        onChange={mockOnChange}
      />
    )

    const modelSelect = screen.getAllByRole('combobox')[1]
    const options = Array.from(modelSelect.querySelectorAll('option')).map(
      (opt) => opt.value
    )

    expect(options).toContain('claude-3-5-sonnet-20240620')
    expect(options).toContain('claude-3-opus-20240229')
    expect(options).toContain('claude-3-sonnet-20240229')
    expect(options).toContain('claude-3-haiku-20240307')
  })

  it('onChange called with (provider, name) signature', async () => {
    const user = userEvent.setup()

    render(
      <ModelSelector
        value={{ provider: 'openai', name: 'gpt-4o-mini' }}
        onChange={mockOnChange}
      />
    )

    const providerSelect = screen.getAllByRole('combobox')[0]

    await user.selectOptions(providerSelect, 'anthropic')

    expect(mockOnChange).toHaveBeenCalledTimes(1)
    expect(mockOnChange.mock.calls[0]).toHaveLength(2)
    expect(typeof mockOnChange.mock.calls[0][0]).toBe('string')
    expect(typeof mockOnChange.mock.calls[0][1]).toBe('string')
  })
})
