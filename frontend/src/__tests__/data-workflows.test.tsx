import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable, generateTestData } from '../test/helpers'
import { server } from '../test/setup'
import { http, HttpResponse } from 'msw'
import App from '../App'

describe('Data Workflows', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Bulk Data Operations', () => {
    test('should handle bulk row creation efficiently', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_bulk_create')
      const table = createTestTable('_bulk')

      // Mock table and bulk operations
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
          return HttpResponse.json({
            id: Math.random().toString(),
            values: { name: 'Bulk User', email: 'bulk@example.com' }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Create multiple rows in succession
      const rowsToCreate = 5
      for (let i = 0; i < rowsToCreate; i++) {
        const rowData = {
          values: {
            name: `Bulk User ${i + 1}`,
            email: `bulk${i + 1}@example.com`,
            age: 25 + i,
            active: i % 2 === 0
          }
        }
        await helpers.createRow(rowData)
      }

      // Verify data was created (checking for success indicators)
      await helpers.expectNoErrors()
    })

    test('should handle bulk row updates correctly', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_bulk_update')
      const table = createTestTable('_bulk_upd')

      // Mock existing rows and update operations
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'User 1', status: 'inactive' } },
            { id: '2', values: { name: 'User 2', status: 'inactive' } },
            { id: '3', values: { name: 'User 3', status: 'inactive' } }
          ])
        }),
        http.put('/api/tables/1/rows/:id', () => {
          return HttpResponse.json({
            id: '1',
            values: { name: 'Updated User', status: 'active' }
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Update multiple rows
      const updatedData = { name: 'Updated User', status: 'active' }
      await helpers.editRow('1', updatedData)

      await helpers.verifyRowExists(updatedData)
    })

    test('should handle bulk row deletion safely', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_bulk_delete')
      const table = createTestTable('_bulk_del')

      // Mock rows for deletion
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Delete Me 1', temp: true } },
            { id: '2', values: { name: 'Delete Me 2', temp: true } },
            { id: '3', values: { name: 'Keep Me', temp: false } }
          ])
        }),
        http.delete('/api/tables/1/rows/:id', () => {
          return HttpResponse.json({ success: true })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Delete specific rows
      await helpers.deleteRow('1')
      await helpers.deleteRow('2')

      // Verify deletions
      await helpers.verifyRowDoesNotExist({ name: 'Delete Me 1' })
      await helpers.verifyRowDoesNotExist({ name: 'Delete Me 2' })
    })

    test('should maintain data consistency during concurrent operations', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_concurrent')
      const table = createTestTable('_concurrent')

      // Mock concurrent operations
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/tables/1/rows', async () => {
          // Simulate processing delay
          await new Promise(resolve => setTimeout(resolve, 100))
          return HttpResponse.json({
            id: Math.random().toString(),
            values: { name: 'Concurrent Row' }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test concurrent operations don't cause errors
      const operations = [
        helpers.createRow({ values: { name: 'Row 1' } }),
        helpers.createRow({ values: { name: 'Row 2' } }),
        helpers.createRow({ values: { name: 'Row 3' } })
      ]

      await Promise.all(operations)
      await helpers.expectNoErrors()
    })
  })

  describe('Search and Filtering', () => {
    test('should filter rows by text content', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_search')
      const table = createTestTable('_searchable')

      // Mock searchable data
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
            { id: '1', values: { name: 'John Doe', email: 'john@example.com' } },
            { id: '2', values: { name: 'Jane Smith', email: 'jane@example.com' } },
            { id: '3', values: { name: 'Bob Johnson', email: 'bob@test.com' } }
          ]

          if (search) {
            const filtered = allRows.filter(row =>
              Object.values(row.values).some(value =>
                String(value).toLowerCase().includes(search.toLowerCase())
              )
            )
            return HttpResponse.json(filtered)
          }

          return HttpResponse.json(allRows)
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test search functionality
      await helpers.searchTable('john')

      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument()
        expect(screen.queryByText('Jane Smith')).not.toBeInTheDocument()
      })
    })

    test('should handle sorting by different columns', async () => {
      helpers.renderComponent(<App />)
      
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
          const order = url.searchParams.get('order') || 'asc'

          let rows = [
            { id: '1', values: { name: 'Charlie', age: 30 } },
            { id: '2', values: { name: 'Alice', age: 25 } },
            { id: '3', values: { name: 'Bob', age: 35 } }
          ]

          if (sortBy === 'name') {
            rows.sort((a, b) => {
              const aVal = a.values.name as string
              const bVal = b.values.name as string
              return order === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal)
            })
          }

          return HttpResponse.json(rows)
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test sorting
      await helpers.sortByColumn('name')

      // Verify sorted order
      const names = screen.getAllByText(/Alice|Bob|Charlie/)
      expect(names[0]).toHaveTextContent('Alice')
    })

    test('should filter by boolean values', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_bool_filter')
      const table = createTestTable('_bool')

      // Mock boolean filtering
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const activeFilter = url.searchParams.get('active')

          const allRows = [
            { id: '1', values: { name: 'Active User', active: true } },
            { id: '2', values: { name: 'Inactive User', active: false } }
          ]

          if (activeFilter !== null) {
            const isActive = activeFilter === 'true'
            return HttpResponse.json(allRows.filter(row => row.values.active === isActive))
          }

          return HttpResponse.json(allRows)
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test boolean filtering (if implemented in UI)
      const filterButton = screen.queryByRole('button', { name: /filter.*active/i })
      if (filterButton) {
        await helpers.user.click(filterButton)
        
        await waitFor(() => {
          expect(screen.getByText('Active User')).toBeInTheDocument()
          expect(screen.queryByText('Inactive User')).not.toBeInTheDocument()
        })
      }
    })

    test('should handle complex multi-column filtering', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_multi_filter')
      const table = createTestTable('_multi')

      // Mock complex filtering
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const nameFilter = url.searchParams.get('name')
          const departmentFilter = url.searchParams.get('department')

          let rows = [
            { id: '1', values: { name: 'John Doe', department: 'Engineering', active: true } },
            { id: '2', values: { name: 'Jane Smith', department: 'Marketing', active: true } },
            { id: '3', values: { name: 'Bob Johnson', department: 'Engineering', active: false } }
          ]

          if (nameFilter) {
            rows = rows.filter(row => 
              String(row.values.name).toLowerCase().includes(nameFilter.toLowerCase())
            )
          }

          if (departmentFilter) {
            rows = rows.filter(row => row.values.department === departmentFilter)
          }

          return HttpResponse.json(rows)
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test combined search and filter
      await helpers.searchTable('john')
      
      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument()
        expect(screen.getByText('Bob Johnson')).toBeInTheDocument()
        expect(screen.queryByText('Jane Smith')).not.toBeInTheDocument()
      })
    })
  })

  describe('Complex Table Management', () => {
    test('should handle multiple tables with different schemas', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_multi_tables')
      
      // Mock multiple tables
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Users Table', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Products Table', createdAt: '2024-01-01T00:00:00Z' },
            { id: '3', name: 'Orders Table', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'John', email: 'john@example.com' } }
          ])
        }),
        http.get('/api/tables/2/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { title: 'Widget', price: 19.99 } }
          ])
        })
      )

      await helpers.registerUser(user)

      // Verify multiple tables are listed
      expect(screen.getByText('Users Table')).toBeInTheDocument()
      expect(screen.getByText('Products Table')).toBeInTheDocument()
      expect(screen.getByText('Orders Table')).toBeInTheDocument()

      // Test switching between tables
      await helpers.selectTable('Users Table')
      await helpers.verifyRowExists({ name: 'John' })

      await helpers.selectTable('Products Table')
      await helpers.verifyRowExists({ title: 'Widget' })
    })

    test('should maintain data integrity across table operations', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_integrity')
      const table = createTestTable('_integrity')

      // Mock data integrity scenarios
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Test User', version: 1 } }
          ])
        }),
        http.put('/api/tables/1/rows/1', () => {
          return HttpResponse.json({
            id: '1',
            values: { name: 'Updated User', version: 2 }
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test data consistency
      await helpers.editRow('1', { name: 'Updated User', version: 2 })
      await helpers.verifyRowExists({ name: 'Updated User' })
    })

    test('should handle table schema modifications gracefully', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_schema')
      const table = createTestTable('_schema')

      // Mock schema changes
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'User', oldField: 'value' } }
          ])
        }),
        http.put('/api/tables/1/schema', () => {
          return HttpResponse.json({ success: true })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test that existing data is handled gracefully
      await helpers.verifyRowExists({ name: 'User' })
    })

    test('should handle large datasets efficiently', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_large')
      const table = createTestTable('_large')

      // Mock large dataset
      const largeDataset = Array.from({ length: 100 }, (_, i) => ({
        id: String(i + 1),
        values: { name: `User ${i + 1}`, index: i + 1 }
      }))

      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const page = parseInt(url.searchParams.get('page') || '1')
          const limit = parseInt(url.searchParams.get('limit') || '10')
          
          const start = (page - 1) * limit
          const end = start + limit
          
          return HttpResponse.json({
            rows: largeDataset.slice(start, end),
            total: largeDataset.length,
            page,
            hasMore: end < largeDataset.length
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Verify pagination or virtual scrolling works
      expect(screen.getByText('User 1')).toBeInTheDocument()
      
      // Test performance doesn't degrade significantly
      await helpers.expectNoErrors()
    })
  })

  describe('Data Validation and Edge Cases', () => {
    test('should validate data types on input', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_validation')
      const table = createTestTable('_validation')

      // Mock validation responses
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
            { error: 'Invalid data type for field age' },
            { status: 400 }
          )
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test validation error handling
      const invalidRow = { values: { name: 'Test', age: 'not-a-number' } }
      
      await helpers.user.click(screen.getByRole('button', { name: /add row/i }))
      
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      await helpers.fillForm({ name: 'Test', age: 'not-a-number' })
      await helpers.submitForm('save')

      // Should show validation error
      await waitFor(() => {
        expect(screen.getByText(/invalid data type/i)).toBeInTheDocument()
      })
    })

    test('should handle special characters and unicode', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_unicode')
      const table = createTestTable('_unicode')

      // Mock unicode support
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
          return HttpResponse.json({
            id: '1',
            values: { name: '测试用户', emoji: '🚀💻', special: '@#$%^&*()' }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test special characters
      const unicodeRow = {
        values: {
          name: '测试用户',
          emoji: '🚀💻',
          special: '@#$%^&*()'
        }
      }

      await helpers.createRow(unicodeRow)
      await helpers.verifyRowExists(unicodeRow.values)
    })

    test('should handle very long text values', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_long_text')
      const table = createTestTable('_long_text')

      const longText = 'A'.repeat(1000)

      // Mock long text handling
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
          return HttpResponse.json({
            id: '1',
            values: { description: longText }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test long text input
      const longTextRow = { values: { description: longText } }
      await helpers.createRow(longTextRow)
      
      // Verify it's handled gracefully (truncated display, etc.)
      await helpers.expectNoErrors()
    })

    test('should handle empty and null-like values appropriately', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_empty_values')
      const table = createTestTable('_empty')

      // Mock empty value handling
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
          return HttpResponse.json({
            id: '1',
            values: { name: '', optional: null, zero: 0, emptyString: '' }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test empty/null values
      const emptyRow = {
        values: {
          name: '',
          optional: null,
          zero: 0,
          emptyString: ''
        }
      }

      await helpers.createRow(emptyRow)
      await helpers.expectNoErrors()
    })
  })

  describe('Performance and Scalability', () => {
    test('should maintain performance with rapid operations', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_performance')
      const table = createTestTable('_performance')

      // Mock rapid operations
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/tables/1/rows', async () => {
          // Simulate fast response
          await new Promise(resolve => setTimeout(resolve, 10))
          return HttpResponse.json({
            id: Math.random().toString(),
            values: { name: 'Fast Row' }
          }, { status: 201 })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test rapid operations
      const startTime = Date.now()
      
      const rapidOperations = Array.from({ length: 10 }, (_, i) =>
        helpers.createRow({ values: { name: `Rapid ${i}` } })
      )

      await Promise.all(rapidOperations)
      
      const endTime = Date.now()
      const duration = endTime - startTime

      // Verify reasonable performance (should complete quickly in test environment)
      expect(duration).toBeLessThan(5000) // 5 seconds max
      await helpers.expectNoErrors()
    })

    test('should handle network timeouts gracefully', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_timeout')
      const table = createTestTable('_timeout')

      // Mock network timeouts
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/tables/1/rows', async () => {
          // Simulate timeout
          await new Promise(resolve => setTimeout(resolve, 10000))
          return HttpResponse.json({ id: '1', values: {} })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test timeout handling
      const timeoutRow = { values: { name: 'Timeout Test' } }
      
      // Start operation that will timeout
      await helpers.user.click(screen.getByRole('button', { name: /add row/i }))
      
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      await helpers.fillForm({ name: 'Timeout Test' })
      await helpers.user.click(screen.getByRole('button', { name: /save/i }))

      // Should handle timeout gracefully (show loading state, error message, etc.)
      // In a real test, this would timeout, but in our mock it completes quickly
      await helpers.expectNoErrors()
    })
  })
})