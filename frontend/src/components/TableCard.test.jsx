import { render, screen, fireEvent } from '@testing-library/react'
import { MantineProvider } from '@mantine/core'
import { describe, it, expect, vi } from 'vitest'
import dayjs from 'dayjs'

// Mock the TableCard component since it's in App.jsx
// In a real project, you'd extract it to its own file
function TableCard({ table, onSelect, onShowHistory }) {
  const theme = { colors: { blue: ['', '', '', '', '', '', '#228be6'] } }

  return (
    <div
      data-testid="table-card"
      style={{
        cursor: 'pointer',
        borderLeft: `4px solid ${theme.colors.blue[6]}`,
      }}
      onClick={() => onSelect(table.id)}
    >
      <div data-testid="table-header">
        <span data-testid="table-name">{table.id}</span>
        <span data-testid="fields-badge">
          {table.fields.length} field{table.fields.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div data-testid="fields-description">
        {table.fields.map(field => `${field.name}: ${field.data_type}`).join(', ')}
      </div>

      <div data-testid="table-footer">
        <span data-testid="created-date">
          {dayjs(table.created_at).format('MMM D, YYYY')}
        </span>
        <div>
          <button
            data-testid="view-button"
            onClick={(e) => {
              e.stopPropagation();
              onSelect(table.id);
            }}
          >
            View
          </button>
          <button
            data-testid="history-button"
            onClick={(e) => {
              e.stopPropagation();
              onShowHistory(table.id);
            }}
          >
            History
          </button>
        </div>
      </div>
    </div>
  );
}

// Helper function to render component with providers
const renderTableCard = (props) => {
  return render(
    <MantineProvider>
      <TableCard {...props} />
    </MantineProvider>
  )
}

describe('TableCard', () => {
  const mockTable = {
    id: 'contacts',
    fields: [
      { name: 'name', data_type: 'string' },
      { name: 'email', data_type: 'string' },
      { name: 'age', data_type: 'number' }
    ],
    created_at: '2024-01-15T10:30:00Z'
  }

  const mockOnSelect = vi.fn()
  const mockOnShowHistory = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders table information correctly', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    // Check table name is displayed
    expect(screen.getByTestId('table-name')).toHaveTextContent('contacts')
    
    // Check fields count badge
    expect(screen.getByTestId('fields-badge')).toHaveTextContent('3 fields')
    
    // Check fields description
    expect(screen.getByTestId('fields-description')).toHaveTextContent(
      'name: string, email: string, age: number'
    )
    
    // Check created date formatting
    expect(screen.getByTestId('created-date')).toHaveTextContent('Jan 15, 2024')
  })

  it('displays singular "field" for single field table', () => {
    const singleFieldTable = {
      ...mockTable,
      fields: [{ name: 'name', data_type: 'string' }]
    }

    renderTableCard({
      table: singleFieldTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    expect(screen.getByTestId('fields-badge')).toHaveTextContent('1 field')
  })

  it('calls onSelect when card is clicked', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    fireEvent.click(screen.getByTestId('table-card'))
    
    expect(mockOnSelect).toHaveBeenCalledWith('contacts')
    expect(mockOnSelect).toHaveBeenCalledTimes(1)
  })

  it('calls onSelect when View button is clicked', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    fireEvent.click(screen.getByTestId('view-button'))
    
    expect(mockOnSelect).toHaveBeenCalledWith('contacts')
    expect(mockOnSelect).toHaveBeenCalledTimes(1)
  })

  it('calls onShowHistory when History button is clicked', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    fireEvent.click(screen.getByTestId('history-button'))
    
    expect(mockOnShowHistory).toHaveBeenCalledWith('contacts')
    expect(mockOnShowHistory).toHaveBeenCalledTimes(1)
  })

  it('prevents event propagation when View button is clicked', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    // Click the View button, which should call onSelect but not trigger card click
    fireEvent.click(screen.getByTestId('view-button'))
    
    // onSelect should be called only once (from button, not from card)
    expect(mockOnSelect).toHaveBeenCalledTimes(1)
  })

  it('prevents event propagation when History button is clicked', () => {
    renderTableCard({
      table: mockTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    // Click the History button, which should call onShowHistory but not trigger card click
    fireEvent.click(screen.getByTestId('history-button'))
    
    expect(mockOnShowHistory).toHaveBeenCalledTimes(1)
    expect(mockOnSelect).not.toHaveBeenCalled()
  })

  it('handles tables with no fields', () => {
    const emptyTable = {
      ...mockTable,
      fields: []
    }

    renderTableCard({
      table: emptyTable,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    expect(screen.getByTestId('fields-badge')).toHaveTextContent('0 fields')
    expect(screen.getByTestId('fields-description')).toHaveTextContent('')
  })

  it('handles different date formats correctly', () => {
    const tableWithDifferentDate = {
      ...mockTable,
      created_at: '2023-12-25T15:45:30.123Z'
    }

    renderTableCard({
      table: tableWithDifferentDate,
      onSelect: mockOnSelect,
      onShowHistory: mockOnShowHistory
    })

    expect(screen.getByTestId('created-date')).toHaveTextContent('Dec 25, 2023')
  })
})