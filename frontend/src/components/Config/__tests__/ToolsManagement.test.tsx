import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { ToolsManagement } from '../ToolsManagement'

describe('ToolsManagement', () => {
  const mockOnChange = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders 4 tools with descriptions', () => {
    render(<ToolsManagement enabledTools={[]} onChange={mockOnChange} />)

    expect(screen.getByText('File Operations')).toBeInTheDocument()
    expect(screen.getByText('Web Search')).toBeInTheDocument()
    expect(screen.getByText('Code Execution')).toBeInTheDocument()
    expect(screen.getByText('Terminal Access')).toBeInTheDocument()

    expect(
      screen.getByText('Read, write, and modify files in the workspace')
    ).toBeInTheDocument()
    expect(screen.getByText('Search the web for information')).toBeInTheDocument()
    expect(screen.getByText('Execute code snippets (Python, JavaScript, etc.)')).toBeInTheDocument()
    expect(screen.getByText('Run shell commands in the workspace')).toBeInTheDocument()
  })

  it('clicking tool card toggles selection', async () => {
    const user = userEvent.setup()

    render(<ToolsManagement enabledTools={[]} onChange={mockOnChange} />)

    const fileOpsCard = screen.getByText('File Operations').closest('div')

    if (!fileOpsCard) throw new Error('Card not found')

    await user.click(fileOpsCard)

    expect(mockOnChange).toHaveBeenCalledWith(['file_ops'])
  })

  it('checkbox change toggles selection', async () => {
    const user = userEvent.setup()

    render(<ToolsManagement enabledTools={[]} onChange={mockOnChange} />)

    const checkboxes = screen.getAllByRole('checkbox')
    const fileOpsCheckbox = checkboxes[0]

    await user.click(fileOpsCheckbox)

    expect(mockOnChange).toHaveBeenCalledWith(['file_ops'])
  })

  it('multiple tools can be selected', async () => {
    const user = userEvent.setup()

    render(<ToolsManagement enabledTools={['file_ops']} onChange={mockOnChange} />)

    const checkboxes = screen.getAllByRole('checkbox')
    const webSearchCheckbox = checkboxes[1]

    await user.click(webSearchCheckbox)

    expect(mockOnChange).toHaveBeenCalledWith(['file_ops', 'web_search'])
  })

  it('onChange called with updated tools array', async () => {
    const user = userEvent.setup()

    render(
      <ToolsManagement enabledTools={['file_ops', 'web_search']} onChange={mockOnChange} />
    )

    const checkboxes = screen.getAllByRole('checkbox')
    const fileOpsCheckbox = checkboxes[0]

    await user.click(fileOpsCheckbox)

    expect(mockOnChange).toHaveBeenCalledWith(['web_search'])
  })

  it('disabled prop prevents toggling', async () => {
    const user = userEvent.setup()

    render(<ToolsManagement enabledTools={[]} onChange={mockOnChange} disabled={true} />)

    const fileOpsCard = screen.getByText('File Operations').closest('div')
    const checkboxes = screen.getAllByRole('checkbox')
    const fileOpsCheckbox = checkboxes[0]

    if (!fileOpsCard) throw new Error('Card not found')

    await user.click(fileOpsCard)
    await user.click(fileOpsCheckbox)

    expect(mockOnChange).not.toHaveBeenCalled()
    expect(fileOpsCheckbox).toBeDisabled()
  })

  it('initial selected tools displayed correctly', () => {
    render(
      <ToolsManagement
        enabledTools={['file_ops', 'terminal']}
        onChange={mockOnChange}
      />
    )

    const checkboxes = screen.getAllByRole('checkbox')
    const fileOpsCheckbox = checkboxes[0]
    const webSearchCheckbox = checkboxes[1]
    const terminalCheckbox = checkboxes[3]

    expect(fileOpsCheckbox).toBeChecked()
    expect(webSearchCheckbox).not.toBeChecked()
    expect(terminalCheckbox).toBeChecked()
  })

  it('empty tools array handled', () => {
    render(<ToolsManagement enabledTools={[]} onChange={mockOnChange} />)

    const allCheckboxes = screen.getAllByRole('checkbox')

    allCheckboxes.forEach((checkbox) => {
      expect(checkbox).not.toBeChecked()
    })
  })
})
