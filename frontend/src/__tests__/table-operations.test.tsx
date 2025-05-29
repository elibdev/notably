import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable } from '../test/helpers'
import { server } from '../test/setup'
import { http, HttpResponse } from 'msw'
import App from '../App'

describe('Table Operations', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Table Management', () => {
    test('should create a new table successfully', async () => {
      render(<App />)
      
      const user = createTestUser('_table_create')
      await helpers.registerUser(user)
      
      const table = createTestTable('_new')
      await helpers.createTable(table)
      
      await helpers.verifyTableExists(table.name)
    })

    test('should list existing tables', async () => {
      render(<App />)
      
      const user = createTestUser('_table_list')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Users Table', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Products Table', createdAt: '2024-01-02T00:00:00Z' }
          ])
        })
      )
      
      await helpers.registerUser(user)
      
      expect(screen.getByText('Users Table')).toBeInTheDocument()
      expect(screen.getByText('Products Table')).toBeInTheDocument()
    })

    test('should delete a table', async () => {
      render(<App />)
      
      const user = createTestUser('_table_delete')
      const table = createTestTable('_delete')
      
      server.use(
        http.delete('/api/tables/:id', () => {
          return HttpResponse.json({ success: true })
        }),
        http.get('/api/tables', () => {
          return HttpResponse.json([])
        })
      )
      
      await helpers.registerUser(user)
      await helpers.createTable(table)
      
      const deleteButton = screen.getByRole('button', { name: /delete.*table/i })
      await helpers.user.click(deleteButton)
      
      const confirmButton = screen.getByRole('button', { name: /confirm|delete|yes/i })
      await helpers.user.click(confirmButton)
      
      await waitFor(() => {
        expect(screen.queryByText(table.name)).not.toBeInTheDocument()
      })
    })

    test('should navigate to table details', async () => {
      render(<App />)
      
      const user = createTestUser('_table_nav')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Test Navigation Table', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Row 1', value: 'Value 1' } }
          ])
        })
      )
      
      await helpers.registerUser(user)
      
      await helpers.selectTable('Test Navigation Table')
      
      expect(screen.getByText('Test Navigation Table')).toBeInTheDocument()
      expect(screen.getByText('Row 1')).toBeInTheDocument()
    })
  })

  describe('Row Operations', () => {
    test('should create a new row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_create')
      const table = createTestTable('_with_rows')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      const rowData = helpers.generateTestRow('_new')
      await helpers.createRow(rowData)
      
      await helpers.verifyRowExists(rowData.values)
    })

    test('should edit an existing row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_edit')
      const table = createTestTable('_editable')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Original Name', value: 'Original Value' } }
          ])
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      const newData = { name: 'Updated Name', value: 'Updated Value' }
      await helpers.editRow('1', newData)
      
      await helpers.verifyRowExists(newData)
    })

    test('should delete a row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_delete')
      const table = createTestTable('_deletable')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Row to Delete', value: 'Delete Me' } }
          ])
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      await helpers.deleteRow('1')
      
      await helpers.verifyRowDoesNotExist({ name: 'Row to Delete', value: 'Delete Me' })
    })
  })

  describe('Search Functionality', () => {
    test('should search rows in table', async () => {
      render(<App />)
      
      const user = createTestUser('_search')
      const table = createTestTable('_searchable')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const search = url.searchParams.get('search')
          const allRows = [
            { id: '1', values: { name: 'Apple Product', category: 'Electronics' } },
            { id: '2', values: { name: 'Banana Snack', category: 'Food' } },
            { id: '3', values: { name: 'Apple Juice', category: 'Beverages' } }
          ]
          
          const filteredRows = search 
            ? allRows.filter(row => 
                Object.values(row.values).some(value => 
                  String(value).toLowerCase().includes(search.toLowerCase())
                )
              )
            : allRows
          
          return HttpResponse.json(filteredRows)
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      expect(screen.getByText('Apple Product')).toBeInTheDocument()
      expect(screen.getByText('Banana Snack')).toBeInTheDocument()
      expect(screen.getByText('Apple Juice')).toBeInTheDocument()
      
      await helpers.searchTable('Apple')
      
      await waitFor(() => {
        expect(screen.getByText('Apple Product')).toBeInTheDocument()
        expect(screen.getByText('Apple Juice')).toBeInTheDocument()
        expect(screen.queryByText('Banana Snack')).not.toBeInTheDocument()
      })
    })

    test('should clear search filter', async () => {
      render(<App />)
      
      const user = createTestUser('_clear_search')
      const table = createTestTable('_clearable')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Item 1', value: 'Value 1' } },
            { id: '2', values: { name: 'Item 2', value: 'Value 2' } }
          ])
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      await helpers.searchTable('Item 1')
      await helpers.clearSearch()
      
      await waitFor(() => {
        expect(screen.getByText('Item 1')).toBeInTheDocument()
        expect(screen.getByText('Item 2')).toBeInTheDocument()
      })
    })
  })

  describe('Error Handling', () => {
    test('should handle table creation errors', async () => {
      render(<App />)
      
      const user = createTestUser('_table_error')
      await helpers.registerUser(user)
      
      server.use(
        http.post('/api/tables', () => {
          return HttpResponse.json(
            { error: 'Table name already exists' },
            { status: 400 }
          )
        })
      )
      
      await helpers.user.click(screen.getByRole('button', { name: /create table/i }))
      
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })
      
      await helpers.user.type(screen.getByPlaceholderText(/table name/i), 'Duplicate Table')
      await helpers.user.click(screen.getByRole('button', { name: /create/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
    })

    test('should handle row creation errors', async () => {
      render(<App />)
      
      const user = createTestUser('_row_error')
      const table = createTestTable('_error_prone')
      
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/tables/1/rows', () => {
          return HttpResponse.json(
            { error: 'Validation failed' },
            { status: 400 }
          )
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      await helpers.user.click(screen.getByRole('button', { name: /add row/i }))
      
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })
      
      await helpers.user.click(screen.getByRole('button', { name: /save/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/validation failed/i)).toBeInTheDocument()
      })
    })
  })
})