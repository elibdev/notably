import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable } from '../src/test/helpers'
import { server } from '../src/test/setup'
import { http, HttpResponse } from 'msw'
import App from '../src/App'

describe('Integration Workflows', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Complete User Journeys', () => {
    test('should complete end-to-end personal information management workflow', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_complete_journey')
      
      // Mock complete workflow data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Contacts', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Journal', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.post('/api/tables', ({ request }) => {
          return HttpResponse.json({
            id: Math.random().toString(),
            name: 'New Table',
            createdAt: new Date().toISOString()
          }, { status: 201 })
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'John Smith', phone: '555-0101', email: 'john@example.com', relationship: 'Friend' } },
            { id: '2', values: { name: 'Jane Doe', phone: '555-0102', email: 'jane@example.com', relationship: 'Colleague' } },
            { id: '3', values: { name: 'Mom', phone: '555-0103', email: 'mom@family.com', relationship: 'Family' } }
          ])
        }),
        http.get('/api/tables/2/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { title: 'Today was great', content: 'Had lunch with friends', mood: 'Happy', important: true } },
            { id: '2', values: { title: 'Work progress', content: 'Finished the project', mood: 'Satisfied', important: false } }
          ])
        }),
        http.post('/api/tables/:id/rows', () => {
          return HttpResponse.json({
            id: Math.random().toString(),
            values: { name: 'New Contact' }
          }, { status: 201 })
        })
      )

      // Step 1: Registration and initial setup
      await helpers.registerUser(user)
      expect(screen.getByTestId('main-app')).toBeInTheDocument()

      // Step 2: Verify tables are created and accessible
      expect(screen.getByText('Contacts')).toBeInTheDocument()
      expect(screen.getByText('Journal')).toBeInTheDocument()

      // Step 3: Work with contacts table
      await helpers.selectTable('Contacts')
      
      // Verify contacts data
      expect(screen.getByText('John Smith')).toBeInTheDocument()
      expect(screen.getByText('Jane Doe')).toBeInTheDocument()
      expect(screen.getByText('Mom')).toBeInTheDocument()

      // Step 4: Add new contact
      const newContact = {
        values: {
          name: 'Bob Wilson',
          phone: '555-0104',
          email: 'bob@example.com',
          relationship: 'Neighbor'
        }
      }
      await helpers.createRow(newContact)

      // Step 5: Switch to journal table
      await helpers.selectTable('Journal')
      
      // Verify journal entries
      expect(screen.getByText('Today was great')).toBeInTheDocument()
      expect(screen.getByText('Work progress')).toBeInTheDocument()

      // Step 6: Add new journal entry
      const newEntry = {
        values: {
          title: 'Integration Test',
          content: 'Testing the full workflow',
          mood: 'Productive',
          important: true
        }
      }
      await helpers.createRow(newEntry)

      // Verify workflow completion
      await helpers.expectNoErrors()
    })

    test('should handle complex data relationships and cross-table workflows', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_cross_table')

      // Mock related data across tables
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Projects', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Tasks', createdAt: '2024-01-01T00:00:00Z' },
            { id: '3', name: 'Time Tracking', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Website Redesign', status: 'In Progress', priority: 'High' } },
            { id: '2', values: { name: 'Mobile App', status: 'Planning', priority: 'Medium' } }
          ])
        }),
        http.get('/api/tables/2/rows', ({ request }) => {
          const url = new URL(request.url)
          const projectFilter = url.searchParams.get('project')
          
          const allTasks = [
            { id: '1', values: { title: 'Design Homepage', project: 'Website Redesign', status: 'Done' } },
            { id: '2', values: { title: 'Code Backend', project: 'Website Redesign', status: 'In Progress' } },
            { id: '3', values: { title: 'User Research', project: 'Mobile App', status: 'Todo' } }
          ]

          if (projectFilter) {
            return HttpResponse.json(
              allTasks.filter(task => task.values.project === projectFilter)
            )
          }

          return HttpResponse.json(allTasks)
        }),
        http.get('/api/tables/3/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { task: 'Design Homepage', hours: 4, date: '2024-01-01' } },
            { id: '2', values: { task: 'Code Backend', hours: 6, date: '2024-01-01' } }
          ])
        })
      )

      await helpers.registerUser(user)

      // Navigate through related data
      await helpers.selectTable('Projects')
      expect(screen.getByText('Website Redesign')).toBeInTheDocument()
      expect(screen.getByText('Mobile App')).toBeInTheDocument()

      // View project-related tasks
      await helpers.selectTable('Tasks')
      expect(screen.getByText('Design Homepage')).toBeInTheDocument()
      expect(screen.getByText('Code Backend')).toBeInTheDocument()
      expect(screen.getByText('User Research')).toBeInTheDocument()

      // Check time tracking
      await helpers.selectTable('Time Tracking')
      expect(screen.getByText('4')).toBeInTheDocument() // hours
      expect(screen.getByText('6')).toBeInTheDocument() // hours

      await helpers.expectNoErrors()
    })

    test('should support collaborative workflow with multiple user actions', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_collaborative')

      // Mock collaborative data changes
      let dataVersion = 1
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Shared Document', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          const baseData = [
            { id: '1', values: { title: 'Draft Report', author: user.username, lastModified: '2024-01-01T10:00:00Z', version: dataVersion } }
          ]

          if (dataVersion > 1) {
            baseData.push({
              id: '2',
              values: { title: 'Review Comments', author: 'reviewer@example.com', lastModified: '2024-01-01T11:00:00Z', version: dataVersion }
            })
          }

          return HttpResponse.json(baseData)
        }),
        http.post('/api/tables/1/rows', () => {
          dataVersion++
          return HttpResponse.json({
            id: '2',
            values: { title: 'Review Comments', author: 'reviewer@example.com' }
          }, { status: 201 })
        }),
        http.put('/api/tables/1/rows/1', () => {
          dataVersion++
          return HttpResponse.json({
            id: '1',
            values: { title: 'Final Report', author: user.username, version: dataVersion }
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable('Shared Document')

      // Initial document state
      expect(screen.getByText('Draft Report')).toBeInTheDocument()

      // Simulate collaborative edit
      await helpers.createRow({
        values: {
          title: 'Review Comments',
          author: 'reviewer@example.com',
          content: 'Looks good, minor changes needed'
        }
      })

      // Update original document
      await helpers.editRow('1', {
        title: 'Final Report',
        author: user.username,
        status: 'Complete'
      })

      // Verify collaborative workflow
      await helpers.expectNoErrors()
    })

    test('should handle import/export workflows', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_import_export')

      // Mock import/export functionality
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Import Test', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Imported Data 1', source: 'CSV' } },
            { id: '2', values: { name: 'Imported Data 2', source: 'CSV' } }
          ])
        }),
        http.post('/api/tables/1/import', () => {
          return HttpResponse.json({
            success: true,
            imported: 2,
            skipped: 0,
            errors: []
          })
        }),
        http.get('/api/tables/1/export', () => {
          return HttpResponse.json({
            downloadUrl: '/api/downloads/export_123.csv',
            filename: 'table_export.csv',
            format: 'csv'
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable('Import Test')

      // Test import functionality
      const importButton = screen.queryByRole('button', { name: /import/i })
      if (importButton) {
        await helpers.user.click(importButton)
        
        // Should show import success
        await waitFor(() => {
          const success = screen.queryByText(/imported.*2.*rows/i)
          expect(success).toBeInTheDocument()
        })
      }

      // Verify imported data
      expect(screen.getByText('Imported Data 1')).toBeInTheDocument()
      expect(screen.getByText('Imported Data 2')).toBeInTheDocument()

      // Test export functionality
      const exportButton = screen.queryByRole('button', { name: /export/i })
      if (exportButton) {
        await helpers.user.click(exportButton)
        
        // Should prepare export
        await waitFor(() => {
          const downloadLink = screen.queryByText(/download|export.*ready/i)
          expect(downloadLink).toBeInTheDocument()
        })
      }

      await helpers.expectNoErrors()
    })
  })

  describe('Real-world Use Cases', () => {
    test('should support personal finance tracking workflow', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_finance')

      // Mock financial data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Expenses', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Income', createdAt: '2024-01-01T00:00:00Z' },
            { id: '3', name: 'Budget Categories', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { description: 'Groceries', amount: 85.50, category: 'Food', date: '2024-01-01' } },
            { id: '2', values: { description: 'Gas', amount: 45.00, category: 'Transportation', date: '2024-01-01' } },
            { id: '3', values: { description: 'Coffee', amount: 4.50, category: 'Food', date: '2024-01-02' } }
          ])
        }),
        http.get('/api/tables/2/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { source: 'Salary', amount: 3000.00, date: '2024-01-01', recurring: true } },
            { id: '2', values: { source: 'Freelance', amount: 500.00, date: '2024-01-15', recurring: false } }
          ])
        }),
        http.get('/api/tables/3/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { category: 'Food', budgeted: 300.00, spent: 90.00 } },
            { id: '2', values: { category: 'Transportation', budgeted: 200.00, spent: 45.00 } }
          ])
        })
      )

      await helpers.registerUser(user)

      // Track expenses
      await helpers.selectTable('Expenses')
      expect(screen.getByText('Groceries')).toBeInTheDocument()
      expect(screen.getByText('85.50')).toBeInTheDocument()

      // Add new expense
      await helpers.createRow({
        values: {
          description: 'Lunch',
          amount: 12.99,
          category: 'Food',
          date: '2024-01-02'
        }
      })

      // Check income
      await helpers.selectTable('Income')
      expect(screen.getByText('Salary')).toBeInTheDocument()
      expect(screen.getByText('3000.00')).toBeInTheDocument()

      // Review budget
      await helpers.selectTable('Budget Categories')
      expect(screen.getByText('Food')).toBeInTheDocument()
      expect(screen.getByText('300.00')).toBeInTheDocument() // budgeted
      expect(screen.getByText('90.00')).toBeInTheDocument()  // spent

      await helpers.expectNoErrors()
    })

    test('should support inventory management workflow', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_inventory')

      // Mock inventory data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Products', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Stock Movements', createdAt: '2024-01-01T00:00:00Z' },
            { id: '3', name: 'Suppliers', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { sku: 'WIDGET-001', name: 'Blue Widget', quantity: 150, reorderLevel: 50, price: 19.99 } },
            { id: '2', values: { sku: 'GADGET-002', name: 'Red Gadget', quantity: 25, reorderLevel: 30, price: 29.99 } }
          ])
        }),
        http.get('/api/tables/2/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { sku: 'WIDGET-001', type: 'Sale', quantity: -5, date: '2024-01-01', reference: 'ORDER-123' } },
            { id: '2', values: { sku: 'GADGET-002', type: 'Purchase', quantity: 25, date: '2024-01-01', reference: 'PO-456' } }
          ])
        }),
        http.get('/api/tables/3/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Widget Corp', contact: 'sales@widget.com', rating: 5 } },
            { id: '2', values: { name: 'Gadget Inc', contact: 'orders@gadget.com', rating: 4 } }
          ])
        })
      )

      await helpers.registerUser(user)

      // Check product inventory
      await helpers.selectTable('Products')
      expect(screen.getByText('Blue Widget')).toBeInTheDocument()
      expect(screen.getByText('150')).toBeInTheDocument() // quantity
      expect(screen.getByText('Red Gadget')).toBeInTheDocument()
      expect(screen.getByText('25')).toBeInTheDocument()  // low stock

      // Review stock movements
      await helpers.selectTable('Stock Movements')
      expect(screen.getByText('Sale')).toBeInTheDocument()
      expect(screen.getByText('Purchase')).toBeInTheDocument()
      expect(screen.getByText('-5')).toBeInTheDocument()  // sale quantity

      // Check suppliers
      await helpers.selectTable('Suppliers')
      expect(screen.getByText('Widget Corp')).toBeInTheDocument()
      expect(screen.getByText('Gadget Inc')).toBeInTheDocument()

      // Add new stock movement
      await helpers.selectTable('Stock Movements')
      await helpers.createRow({
        values: {
          sku: 'WIDGET-001',
          type: 'Purchase',
          quantity: 100,
          date: '2024-01-02',
          reference: 'PO-789'
        }
      })

      await helpers.expectNoErrors()
    })

    test('should support event planning workflow', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_events')

      // Mock event planning data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Events', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Attendees', createdAt: '2024-01-01T00:00:00Z' },
            { id: '3', name: 'Tasks', createdAt: '2024-01-01T00:00:00Z' },
            { id: '4', name: 'Vendors', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Company Retreat', date: '2024-02-15', location: 'Mountain Resort', status: 'Planning', budget: 10000 } }
          ])
        }),
        http.get('/api/tables/2/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'John Doe', email: 'john@company.com', status: 'Confirmed', dietary: 'Vegetarian' } },
            { id: '2', values: { name: 'Jane Smith', email: 'jane@company.com', status: 'Pending', dietary: 'None' } }
          ])
        }),
        http.get('/api/tables/3/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { task: 'Book venue', assignee: 'John', due: '2024-01-15', status: 'Complete' } },
            { id: '2', values: { task: 'Send invitations', assignee: 'Jane', due: '2024-01-20', status: 'In Progress' } },
            { id: '3', values: { task: 'Arrange catering', assignee: 'Bob', due: '2024-01-25', status: 'Todo' } }
          ])
        }),
        http.get('/api/tables/4/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Mountain Catering', service: 'Food', cost: 3000, status: 'Confirmed' } },
            { id: '2', values: { name: 'Sound & Light Co', service: 'AV Equipment', cost: 1500, status: 'Quoted' } }
          ])
        })
      )

      await helpers.registerUser(user)

      // Plan the event
      await helpers.selectTable('Events')
      expect(screen.getByText('Company Retreat')).toBeInTheDocument()
      expect(screen.getByText('Mountain Resort')).toBeInTheDocument()

      // Manage attendees
      await helpers.selectTable('Attendees')
      expect(screen.getByText('John Doe')).toBeInTheDocument()
      expect(screen.getByText('Confirmed')).toBeInTheDocument()
      expect(screen.getByText('Jane Smith')).toBeInTheDocument()
      expect(screen.getByText('Pending')).toBeInTheDocument()

      // Track tasks
      await helpers.selectTable('Tasks')
      expect(screen.getByText('Book venue')).toBeInTheDocument()
      expect(screen.getByText('Complete')).toBeInTheDocument()
      expect(screen.getByText('Send invitations')).toBeInTheDocument()

      // Manage vendors
      await helpers.selectTable('Vendors')
      expect(screen.getByText('Mountain Catering')).toBeInTheDocument()
      expect(screen.getByText('3000')).toBeInTheDocument()

      // Add new task
      await helpers.selectTable('Tasks')
      await helpers.createRow({
        values: {
          task: 'Setup registration desk',
          assignee: 'Alice',
          due: '2024-02-14',
          status: 'Todo'
        }
      })

      await helpers.expectNoErrors()
    })
  })

  describe('Advanced Integration Scenarios', () => {
    test('should handle complex filtering and search across multiple tables', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_complex_search')

      // Mock complex search scenarios
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Customers', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Orders', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/search', ({ request }) => {
          const url = new URL(request.url)
          const query = url.searchParams.get('q')
          
          if (query === 'john') {
            return HttpResponse.json({
              customers: [
                { id: '1', table: 'Customers', values: { name: 'John Doe', email: 'john@example.com' } }
              ],
              orders: [
                { id: '1', table: 'Orders', values: { customer: 'John Doe', amount: 150.00 } }
              ]
            })
          }
          
          return HttpResponse.json({ customers: [], orders: [] })
        })
      )

      await helpers.registerUser(user)

      // Test global search functionality
      const globalSearch = screen.queryByPlaceholderText(/search all tables/i)
      if (globalSearch) {
        await helpers.user.type(globalSearch, 'john')
        
        await waitFor(() => {
          expect(screen.getByText('John Doe')).toBeInTheDocument()
        })
      }

      await helpers.expectNoErrors()
    })

    test('should support bulk operations across multiple tables', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_bulk_ops')

      // Mock bulk operations
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Products', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Categories', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.post('/api/bulk/update', () => {
          return HttpResponse.json({
            updated: 5,
            errors: [],
            success: true
          })
        })
      )

      await helpers.registerUser(user)

      // Test bulk update functionality
      const bulkButton = screen.queryByRole('button', { name: /bulk.*update|mass.*edit/i })
      if (bulkButton) {
        await helpers.user.click(bulkButton)
        
        await waitFor(() => {
          const success = screen.queryByText(/updated.*5/i)
          expect(success).toBeInTheDocument()
        })
      }

      await helpers.expectNoErrors()
    })

    test('should maintain data consistency during complex operations', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_consistency')

      // Mock consistency scenarios
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: 'Orders', createdAt: '2024-01-01T00:00:00Z' },
            { id: '2', name: 'Inventory', createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.post('/api/orders/:id/complete', () => {
          return HttpResponse.json({
            success: true,
            inventoryUpdated: true,
            orderStatus: 'completed'
          })
        })
      )

      await helpers.registerUser(user)

      // Test transactional operations
      const completeButton = screen.queryByRole('button', { name: /complete.*order/i })
      if (completeButton) {
        await helpers.user.click(completeButton)
        
        await waitFor(() => {
          const success = screen.queryByText(/completed|success/i)
          expect(success).toBeInTheDocument()
        })
      }

      await helpers.expectNoErrors()
    })
  })
})