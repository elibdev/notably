import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { vi } from 'vitest'
import { server } from './setup'
import { http, HttpResponse } from 'msw'
import React, { ReactElement } from 'react'
import { MantineProvider } from '@mantine/core'

export interface TestUser {
  username: string;
  email: string;
  password: string;
  apiKey?: string;
}

export interface TestTable {
  name: string;
  columns?: Array<{ name: string; dataType: string }>;
}

export interface TestRow {
  id?: string;
  values: Record<string, unknown>;
}

export class TestHelpers {
  public user = userEvent.setup()

  // Component rendering helpers
  renderComponent(component: ReactElement) {
    const TestWrapper = ({ children }: { children: React.ReactNode }) => (
      <MantineProvider>
        {children}
      </MantineProvider>
    )
    
    return render(component, { wrapper: TestWrapper })
  }

  // Authentication helpers
  async registerUser(userData: TestUser): Promise<void> {
    // Mock successful registration
    server.use(
      http.post('/api/auth/register', () => {
        return HttpResponse.json({
          success: true,
          user: { id: '1', username: userData.username, email: userData.email },
          apiKey: 'test-api-key'
        })
      })
    )

    // Find and interact with registration form
    const registerTab = screen.queryByRole('tab', { name: /register/i })
    if (registerTab) {
      await this.user.click(registerTab)
    }

    await this.user.type(screen.getByPlaceholderText('Username'), userData.username)
    await this.user.type(screen.getByPlaceholderText('Email'), userData.email)
    await this.user.type(screen.getByPlaceholderText('Password'), userData.password)
    
    await this.user.click(screen.getByRole('button', { name: /register/i }))
    
    // Wait for successful registration
    await waitFor(() => {
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
    }, { timeout: 5000 })
  }

  async loginUser(userData: TestUser): Promise<void> {
    // Mock successful login
    server.use(
      http.post('/api/auth/login', () => {
        return HttpResponse.json({
          success: true,
          user: { id: '1', username: userData.username, email: userData.email },
          apiKey: 'test-api-key'
        })
      })
    )

    const loginTab = screen.queryByRole('tab', { name: /login/i })
    if (loginTab) {
      await this.user.click(loginTab)
    }

    await this.user.type(screen.getByPlaceholderText('Username'), userData.username)
    await this.user.type(screen.getByPlaceholderText('Password'), userData.password)
    
    await this.user.click(screen.getByRole('button', { name: /login/i }))
    
    await waitFor(() => {
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
    }, { timeout: 5000 })
  }

  async logout(): Promise<void> {
    server.use(
      http.post('/api/auth/logout', () => {
        return HttpResponse.json({ success: true })
      })
    )

    const logoutButton = screen.getByRole('button', { name: /logout/i })
    await this.user.click(logoutButton)
    
    await waitFor(() => {
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  }

  // Table helpers
  async createTable(tableData: TestTable): Promise<void> {
    server.use(
      http.post('/api/tables', () => {
        return HttpResponse.json({
          id: '2',
          name: tableData.name,
          createdAt: new Date().toISOString()
        }, { status: 201 })
      })
    )

    await this.user.click(screen.getByRole('button', { name: /create table/i }))
    
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    
    await this.user.type(screen.getByPlaceholderText(/table name/i), tableData.name)
    
    if (tableData.columns) {
      for (const column of tableData.columns) {
        await this.user.click(screen.getByRole('button', { name: /add column/i }))
        const columnInputs = screen.getAllByPlaceholderText(/column name/i)
        await this.user.type(columnInputs[columnInputs.length - 1], column.name)
      }
    }
    
    await this.user.click(screen.getByRole('button', { name: /create/i }))
    
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  }

  async selectTable(tableName: string): Promise<void> {
    const tableLink = screen.getByRole('link', { name: new RegExp(tableName, 'i') })
    await this.user.click(tableLink)
    
    await waitFor(() => {
      expect(screen.getByText(tableName)).toBeInTheDocument()
    })
  }

  async createRow(rowData: TestRow): Promise<void> {
    server.use(
      http.post('/api/tables/:id/rows', () => {
        return HttpResponse.json({
          id: '2',
          values: rowData.values
        }, { status: 201 })
      })
    )

    await this.user.click(screen.getByRole('button', { name: /add row/i }))
    
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    
    for (const [key, value] of Object.entries(rowData.values)) {
      const input = screen.getByLabelText(new RegExp(key, 'i'))
      await this.user.clear(input)
      await this.user.type(input, String(value))
    }
    
    await this.user.click(screen.getByRole('button', { name: /save/i }))
    
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  }

  async editRow(rowId: string, newData: Record<string, unknown>): Promise<void> {
    server.use(
      http.put(`/api/tables/:tableId/rows/${rowId}`, () => {
        return HttpResponse.json({
          id: rowId,
          values: newData
        })
      })
    )

    const editButton = screen.getByRole('button', { name: /edit.*row/i })
    await this.user.click(editButton)
    
    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })
    
    for (const [key, value] of Object.entries(newData)) {
      const input = screen.getByLabelText(new RegExp(key, 'i'))
      await this.user.clear(input)
      await this.user.type(input, String(value))
    }
    
    await this.user.click(screen.getByRole('button', { name: /save/i }))
    
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  }

  async deleteRow(rowId: string): Promise<void> {
    server.use(
      http.delete(`/api/tables/:tableId/rows/${rowId}`, () => {
        return HttpResponse.json({ success: true })
      })
    )

    const deleteButton = screen.getByRole('button', { name: /delete.*row/i })
    await this.user.click(deleteButton)
    
    // Confirm deletion if confirmation dialog appears
    const confirmButton = screen.queryByRole('button', { name: /confirm|delete|yes/i })
    if (confirmButton) {
      await this.user.click(confirmButton)
    }
    
    await waitFor(() => {
      expect(screen.queryByText(`Row ${rowId}`)).not.toBeInTheDocument()
    })
  }

  // Verification helpers
  async verifyTableExists(tableName: string): Promise<void> {
    await waitFor(() => {
      expect(screen.getByText(tableName)).toBeInTheDocument()
    })
  }

  async verifyRowExists(rowData: Record<string, unknown>): Promise<void> {
    for (const [key, value] of Object.entries(rowData)) {
      await waitFor(() => {
        expect(screen.getByText(String(value))).toBeInTheDocument()
      })
    }
  }

  async verifyRowDoesNotExist(rowData: Record<string, unknown>): Promise<void> {
    for (const [key, value] of Object.entries(rowData)) {
      await waitFor(() => {
        expect(screen.queryByText(String(value))).not.toBeInTheDocument()
      })
    }
  }

  async verifyNotificationMessage(message: string): Promise<void> {
    await waitFor(() => {
      expect(screen.getByText(new RegExp(message, 'i'))).toBeInTheDocument()
    })
  }

  async verifyErrorMessage(message: string): Promise<void> {
    await waitFor(() => {
      expect(screen.getByText(new RegExp(message, 'i'))).toBeInTheDocument()
    })
  }

  async expectNoErrors(): Promise<void> {
    // Check that no error messages are visible
    const errorPatterns = [/error/i, /failed/i, /something went wrong/i]
    for (const pattern of errorPatterns) {
      expect(screen.queryByText(pattern)).not.toBeInTheDocument()
    }
  }

  // Utility helpers
  generateTestUser(suffix: string = ''): TestUser {
    const timestamp = Date.now()
    return {
      username: `testuser${suffix}_${timestamp}`,
      email: `test${suffix}_${timestamp}@example.com`,
      password: 'TestPassword123!'
    }
  }

  generateTestTable(suffix: string = ''): TestTable {
    const timestamp = Date.now()
    return {
      name: `Test Table${suffix} ${timestamp}`,
      columns: [
        { name: 'Name', dataType: 'string' },
        { name: 'Value', dataType: 'string' }
      ]
    }
  }

  generateTestRow(suffix: string = ''): TestRow {
    const timestamp = Date.now()
    return {
      values: {
        name: `Test Row${suffix} ${timestamp}`,
        value: `Test Value${suffix} ${timestamp}`
      }
    }
  }

  // Mock network errors
  simulateNetworkError(endpoint: string): void {
    server.use(
      http.all(endpoint, () => {
        return HttpResponse.error()
      })
    )
  }

  simulateServerError(endpoint: string, status: number = 500): void {
    server.use(
      http.all(endpoint, () => {
        return HttpResponse.json(
          { error: 'Internal server error' },
          { status }
        )
      })
    )
  }

  removeNetworkErrorSimulation(): void {
    server.resetHandlers()
  }

  // Form helpers
  async fillForm(formData: Record<string, string>): Promise<void> {
    for (const [label, value] of Object.entries(formData)) {
      const input = screen.getByLabelText(new RegExp(label, 'i'))
      await this.user.clear(input)
      await this.user.type(input, value)
    }
  }

  async submitForm(buttonText: string = 'submit'): Promise<void> {
    const submitButton = screen.getByRole('button', { name: new RegExp(buttonText, 'i') })
    await this.user.click(submitButton)
  }

  // Accessibility helpers
  async checkFocusVisible(): Promise<void> {
    const focusedElement = document.activeElement
    expect(focusedElement).toBeInTheDocument()
  }

  async checkAriaAttributes(element: HTMLElement, expectedAttributes: Record<string, string>): Promise<void> {
    for (const [attr, value] of Object.entries(expectedAttributes)) {
      expect(element).toHaveAttribute(attr, value)
    }
  }

  // Search and filter helpers
  async searchTable(searchTerm: string): Promise<void> {
    const searchInput = screen.getByPlaceholderText(/search/i)
    await this.user.clear(searchInput)
    await this.user.type(searchInput, searchTerm)
    
    // Wait for search results to update
    await waitFor(() => {
      expect(searchInput).toHaveValue(searchTerm)
    })
  }

  async clearSearch(): Promise<void> {
    const searchInput = screen.getByPlaceholderText(/search/i)
    await this.user.clear(searchInput)
  }

  async sortByColumn(columnName: string): Promise<void> {
    const columnHeader = screen.getByRole('columnheader', { name: new RegExp(columnName, 'i') })
    await this.user.click(columnHeader)
  }
}

// Helper functions
export function createTestUser(suffix: string = ''): TestUser {
  const timestamp = Date.now()
  return {
    username: `testuser${suffix}_${timestamp}`,
    email: `test${suffix}_${timestamp}@example.com`,
    password: 'TestPassword123!'
  }
}

export function createTestTable(suffix: string = ''): TestTable {
  const timestamp = Date.now()
  return {
    name: `Test Table${suffix} ${timestamp}`,
    columns: [
      { name: 'Name', dataType: 'string' },
      { name: 'Value', dataType: 'string' }
    ]
  }
}

export function createCustomTestTable(name: string, columns: Array<{ name: string; dataType: string }>): TestTable {
  return { name, columns }
}

export function generateTestData(count: number): TestRow[] {
  return Array.from({ length: count }, (_, i) => ({
    values: {
      name: `Test Row ${i + 1}`,
      value: `Test Value ${i + 1}`,
      index: i + 1
    }
  }))
}

// Mock data setup helpers
export async function setupTestEnvironment(): Promise<TestUser> {
  const user = createTestUser('_setup')
  
  // Setup default mock responses
  server.use(
    http.get('/api/tables', () => {
      return HttpResponse.json([
        { id: '1', name: 'Default Table', createdAt: '2024-01-01T00:00:00Z' }
      ])
    })
  )
  
  return user
}

export async function setupComplexTestEnvironment(): Promise<{
  user: TestUser;
  tables: TestTable[];
  rows: TestRow[];
}> {
  const user = createTestUser('_complex')
  const tables = [
    createTestTable('_main'),
    createTestTable('_secondary')
  ]
  const rows = generateTestData(5)
  
  // Setup comprehensive mock responses
  server.use(
    http.get('/api/tables', () => {
      return HttpResponse.json(tables.map((table, i) => ({
        id: String(i + 1),
        name: table.name,
        createdAt: '2024-01-01T00:00:00Z'
      })))
    }),
    http.get('/api/tables/:id/rows', () => {
      return HttpResponse.json(rows.map((row, i) => ({
        id: String(i + 1),
        values: row.values
      })))
    })
  )
  
  return { user, tables, rows }
}