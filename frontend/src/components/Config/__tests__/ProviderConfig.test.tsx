import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ProviderConfig } from '../ProviderConfig'

describe('ProviderConfig', () => {
  const mockOnChange = vi.fn()

  const defaultProps = {
    provider: 'openai',
    apiKey: 'sk-test123',
    temperature: 0.7,
    maxTokens: 4000,
    onChange: mockOnChange,
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders all fields with initial values', () => {
    render(<ProviderConfig {...defaultProps} />)

    const apiKeyInput = screen.getByPlaceholderText(/sk-\.\.\./i)
    const temperatureSlider = screen.getByRole('slider')
    const maxTokensInput = screen.getByRole('spinbutton')

    expect(apiKeyInput).toHaveValue('sk-test123')
    expect(temperatureSlider).toHaveValue('0.7')
    expect(maxTokensInput).toHaveValue(4000)
  })

  it('API key show/hide toggle works', async () => {
    const user = userEvent.setup()

    render(<ProviderConfig {...defaultProps} />)

    const apiKeyInput = screen.getByPlaceholderText(/sk-\.\.\./i) as HTMLInputElement
    const toggleButton = screen.getByRole('button', { name: '' })

    expect(apiKeyInput.type).toBe('password')

    await user.click(toggleButton)

    expect(apiKeyInput.type).toBe('text')

    await user.click(toggleButton)

    expect(apiKeyInput.type).toBe('password')
  })

  it('API key input calls onChange with api_key field', async () => {
    const user = userEvent.setup()

    render(<ProviderConfig {...defaultProps} apiKey="" />)

    const apiKeyInput = screen.getByPlaceholderText(/sk-\.\.\./i)

    await user.type(apiKeyInput, 'test')

    expect(mockOnChange).toHaveBeenCalled()
    const firstCall = mockOnChange.mock.calls[0]
    expect(firstCall[0]).toBe('api_key')
    expect(typeof firstCall[1]).toBe('string')
  })

  it('temperature slider updates and calls onChange', async () => {
    const user = userEvent.setup()

    render(<ProviderConfig {...defaultProps} />)

    const temperatureSlider = screen.getByRole('slider')

    fireEvent.change(temperatureSlider, { target: { value: '1.5' } })

    expect(mockOnChange).toHaveBeenCalledWith('temperature', 1.5)
  })

  it('max tokens input updates and calls onChange', () => {
    render(<ProviderConfig {...defaultProps} maxTokens={1000} />)

    const maxTokensInput = screen.getByRole('spinbutton')

    fireEvent.change(maxTokensInput, { target: { value: '5000' } })

    expect(mockOnChange).toHaveBeenCalledWith('max_tokens', 5000)
  })

  it('custom provider shows API endpoint field', () => {
    render(<ProviderConfig {...defaultProps} provider="custom" />)

    const apiEndpointInput = screen.getByPlaceholderText('https://api.openai.com/v1')

    expect(apiEndpointInput).toBeInTheDocument()
    expect(apiEndpointInput).toHaveAttribute('placeholder', 'https://api.openai.com/v1')
  })

  it('non-custom provider hides API endpoint field', () => {
    render(<ProviderConfig {...defaultProps} provider="openai" />)

    const apiEndpointInput = screen.queryByPlaceholderText('https://api.openai.com/v1')

    expect(apiEndpointInput).not.toBeInTheDocument()
  })

  it('disabled prop disables all inputs', () => {
    render(<ProviderConfig {...defaultProps} disabled={true} />)

    const apiKeyInput = screen.getByPlaceholderText('••••••••••••••••')
    const temperatureSlider = screen.getByRole('slider')
    const maxTokensInput = screen.getByRole('spinbutton')
    const toggleButton = screen.getByRole('button', { name: '' })

    expect(apiKeyInput).toBeDisabled()
    expect(temperatureSlider).toBeDisabled()
    expect(maxTokensInput).toBeDisabled()
    expect(toggleButton).toBeDisabled()
  })

  it('temperature displays current value in label', () => {
    render(<ProviderConfig {...defaultProps} temperature={0.9} />)

    expect(screen.getByText(/temperature: 0\.9/i)).toBeInTheDocument()
  })

  it('security note text rendered', () => {
    render(<ProviderConfig {...defaultProps} />)

    expect(
      screen.getByText(/api key is encrypted and never shown in responses/i)
    ).toBeInTheDocument()
  })
})
