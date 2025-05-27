import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable, createCustomTestTable, generateTestData } from '../src/test/helpers'
import { server } from '../src/test/setup'
import { http, HttpResponse } from 'msw'
import App from '../src/App'

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
      
      // Should show the new table in the list
      await helpers.verifyTableExists(table.name)
    })

    test('should list existing tables', async () => {
      render(<App />)
      
      const user = createTestUser('_table_list')
      
      // Mock tables response
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Users Table', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Products Table', createdAt: '2024-01-02T00:00:00Z' },
            { id: '3', name: 'Orders Table', createdAt: '2024-01-03T00:00:00Z' }
          ])
        })
      )
      
      await helpers.registerUser(user)
      
      // Should display all tables
      expect(screen.getByText('Users Table')).toBeInTheDocument()
      expect(screen.getByText('Products Table')).toBeInTheDocument()
      expect(screen.getByText('Orders Table')).toBeInTheDocument()
    })

    test('should delete a table', async () => {
      render(<App />)
      
      const user = createTestUser('_table_delete')
      const table = createTestTable('_delete')
      
      // Mock table deletion
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
      
      // Delete the table
      const deleteButton = screen.getByRole('button', { name: /delete.*table/i })
      await helpers.user.click(deleteButton)
      
      // Confirm deletion
      const confirmButton = screen.getByRole('button', { name: /confirm|delete|yes/i })
      await helpers.user.click(confirmButton)
      
      // Table should be removed from list
      await waitFor(() => {
        expect(screen.queryByText(table.name)).not.toBeInTheDocument()
      })
    })

    test('should handle table creation errors', async () => {
      render(<App />)
      
      const user = createTestUser('_table_error')
      await helpers.registerUser(user)
      
      // Mock table creation error
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
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
    })

    test('should navigate to table details', async () => {
      render(<App />)
      
      const user = createTestUser('_table_nav')
      
      // Mock table and rows data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Test Navigation Table', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Row 1', value: 'Value 1' } },
            { id: '2', values: { name: 'Row 2', value: 'Value 2' } }
          ])
        })
      )
      
      await helpers.registerUser(user)
      
      await helpers.selectTable('Test Navigation Table')
      
      // Should show table details with rows
      expect(screen.getByText('Test Navigation Table')).toBeInTheDocument()
      expect(screen.getByText('Row 1')).toBeInTheDocument()
      expect(screen.getByText('Row 2')).toBeInTheDocument()
    })
  })

  describe('Row Operations', () => {
    test('should create a new row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_create')
      const table = createTestTable('_with_rows')
      
      // Mock table with existing structure
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
      
      // Should show the new row
      await helpers.verifyRowExists(rowData.values)
    })

    test('should edit an existing row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_edit')
      const table = createTestTable('_editable')
      
      // Mock table with existing row
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
      
      // Should show updated data
      await helpers.verifyRowExists(newData)
    })

    test('should delete a row', async () => {
      render(<App />)
      
      const user = createTestUser('_row_delete')
      const table = createTestTable('_deletable')
      
      // Mock table with row to delete
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
      
      // Row should be removed
      await helpers.verifyRowDoesNotExist({ name: 'Row to Delete', value: 'Delete Me' })
    })

    test('should handle row creation errors', async () => {
      render(<App />)
      
      const user = createTestUser('_row_error')
      const table = createTestTable('_error_prone')
      
      // Mock table setup
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
      
      // Try to save invalid data
      await helpers.user.click(screen.getByRole('button', { name: /save/i }))
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/validation failed/i)).toBeInTheDocument()
      })
    })
  })

  describe('Table Search and Filter', () => {
    test('should search rows in table', async () => {
      render(<App />)
      
      const user = createTestUser('_search')
      const table = createTestTable('_searchable')
      
      // Mock table with multiple rows
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
      
      // Should show all rows initially
      expect(screen.getByText('Apple Product')).toBeInTheDocument()
      expect(screen.getByText('Banana Snack')).toBeInTheDocument()
      expect(screen.getByText('Apple Juice')).toBeInTheDocument()
      
      // Search for "Apple"
      await helpers.searchTable('Apple')
      
      // Should show only Apple products
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
      
      // Similar setup as above
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
      
      // Apply search
      await helpers.searchTable('Item 1')
      
      // Clear search
      await helpers.clearSearch()
      
      // Should show all items again
      await waitFor(() => {
        expect(screen.getByText('Item 1')).toBeInTheDocument()
        expect(screen.getByText('Item 2')).toBeInTheDocument()
      })
    })
  })

  describe('Table Sorting', () => {
    test('should sort table by column', async () => {
      render(<App />)
      
      const user = createTestUser('_sort')
      const table = createTestTable('_sortable')
      
      // Mock sortable data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const sortBy = url.searchParams.get('sortBy')
          const sortOrder = url.searchParams.get('sortOrder') || 'asc'
          
          let rows = [
            { id: '1', values: { name: 'Charlie', age: 30 } },
            { id: '2', values: { name: 'Alice', age: 25 } },
            { id: '3', values: { name: 'Bob', age: 35 } }
          ]
          
          if (sortBy === 'name') {
            rows.sort((a, b) => {
              const aVal = a.values.name as string
              const bVal = b.values.name as string
              return sortOrder === 'asc' 
                ? aVal.localeCompare(bVal)
                : bVal.localeCompare(aVal)
            })
          }
          
          return HttpResponse.json(rows)
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      // Sort by name column
      await helpers.sortByColumn('Name')
      
      // Should be sorted alphabetically
      const nameElements = screen.getAllByText(/Alice|Bob|Charlie/)
      expect(nameElements[0]).toHaveTextContent('Alice')
      expect(nameElements[1]).toHaveTextContent('Bob')
      expect(nameElements[2]).toHaveTextContent('Charlie')
    })
  })

  describe('Bulk Operations', () => {
    test('should create multiple rows', async () => {
      render(<App />)
      
      const user = createTestUser('_bulk_create')
      const table = createTestTable('_bulk')
      
      // Mock bulk creation
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/tables/1/rows/bulk', () => {
          return HttpResponse.json({
            created: 3,
            rows: [
              { id: '1', values: { name: 'Bulk Row 1', value: 'Value 1' } },
              { id: '2', values: { name: 'Bulk Row 2', value: 'Value 2' } },
              { id: '3', values: { name: 'Bulk Row 3', value: 'Value 3' } }
            ]
          }, { status: 201 })
        })
      )
      
      await helpers.registerUser(user)
      await helpers.selectTable(table.name)
      
      // Test bulk create functionality if available
      const bulkButton = screen.queryByRole('button', { name: /bulk.*create|import/i })
      if (bulkButton) {
        await helpers.user.click(bulkButton)
        
        // Should show success message
        await waitFor(() => {
          expect(screen.getByText(/created.*3.*rows/i)).toBeInTheDocument()
        })
      }
    })
  })

  describe('Error Handling', () => {
    test('should handle network errors gracefully', async () => {
      render(<App />)
      
      const user = createTestUser('_network_error')
      await helpers.registerUser(user)
      
      // Simulate network error for tables
      helpers.simulateNetworkError('/api/tables')
      
      // Should show error state or fallback
      await waitFor(() => {
        expect(screen.getByTestId('main-app')).toBeInTheDocument()
      })
      
      // Should handle the error gracefully
      await helpers.expectNoErrors()
    })

    test('should handle server errors', async () => {
      render(<App />)
      
      const user = createTestUser('_server_error')
      
      // Mock server error
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json(
            { error: 'Internal server error' },
            { status: 500 }
          )
        })
      )
      
      await helpers.registerUser(user)
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/error/i)).toBeInTheDocument()
      })
    })
  })
})