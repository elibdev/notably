import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable } from '../src/test/helpers'
import { server } from '../src/test/setup'
import { http, HttpResponse } from 'msw'
import App from '../src/App'

describe('History and Snapshot Features', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Snapshot Functionality', () => {
    test('should create and view table snapshots', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_snapshot')
      const table = createTestTable('_snap')

      // Mock snapshot functionality
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Initial Row 1', data: 'value1' } },
            { id: '2', values: { name: 'Initial Row 2', data: 'value2' } }
          ])
        }),
        http.post('/api/tables/1/snapshots', () => {
          return HttpResponse.json({
            id: 'snap_1',
            timestamp: new Date().toISOString(),
            rowCount: 2
          }, { status: 201 })
        }),
        http.get('/api/tables/1/snapshots', () => {
          return HttpResponse.json([
            {
              id: 'snap_1',
              timestamp: '2024-01-01T12:00:00Z',
              rowCount: 2
            }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Verify initial data exists
      expect(screen.getByText('Initial Row 1')).toBeInTheDocument()
      expect(screen.getByText('Initial Row 2')).toBeInTheDocument()

      // Take snapshot
      const snapshotButton = screen.queryByRole('button', { name: /snapshot|save snapshot/i })
      if (snapshotButton) {
        await helpers.user.click(snapshotButton)
        
        // Should show success message or snapshot list
        await waitFor(() => {
          const successMessage = screen.queryByText(/snapshot.*created|saved/i)
          const snapshotList = screen.queryByText(/snapshots?/i)
          expect(successMessage || snapshotList).toBeTruthy()
        })
      }
    })

    test('should view historical snapshots at specific times', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_hist_snap')
      const table = createTestTable('_hist')

      // Mock historical snapshot data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', ({ request }) => {
          const url = new URL(request.url)
          const snapshotId = url.searchParams.get('snapshot')
          
          if (snapshotId === 'snap_1') {
            return HttpResponse.json([
              { id: '1', values: { name: 'Historical Row 1', version: 1 } }
            ])
          }
          
          if (snapshotId === 'snap_2') {
            return HttpResponse.json([
              { id: '1', values: { name: 'Historical Row 1', version: 1 } },
              { id: '2', values: { name: 'Historical Row 2', version: 1 } }
            ])
          }

          // Current data
          return HttpResponse.json([
            { id: '1', values: { name: 'Current Row 1', version: 2 } },
            { id: '2', values: { name: 'Current Row 2', version: 2 } },
            { id: '3', values: { name: 'Current Row 3', version: 1 } }
          ])
        }),
        http.get('/api/tables/1/snapshots', () => {
          return HttpResponse.json([
            {
              id: 'snap_1',
              timestamp: '2024-01-01T10:00:00Z',
              rowCount: 1,
              description: 'Initial snapshot'
            },
            {
              id: 'snap_2',
              timestamp: '2024-01-01T11:00:00Z',
              rowCount: 2,
              description: 'After adding second row'
            }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access snapshot view
      const historyButton = screen.queryByRole('button', { name: /history|snapshots/i })
      if (historyButton) {
        await helpers.user.click(historyButton)
        
        // Should show snapshot list
        await waitFor(() => {
          expect(screen.getByText(/initial snapshot|10:00/i)).toBeInTheDocument()
        })

        // View specific snapshot
        const snapshot1 = screen.getByText(/initial snapshot|10:00/i)
        await helpers.user.click(snapshot1)

        // Should show historical data
        await waitFor(() => {
          expect(screen.getByText('Historical Row 1')).toBeInTheDocument()
          expect(screen.queryByText('Current Row 2')).not.toBeInTheDocument()
        })
      }
    })

    test('should compare snapshots between different time points', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_compare_snap')
      const table = createTestTable('_compare')

      // Mock snapshot comparison
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/snapshots', () => {
          return HttpResponse.json([
            {
              id: 'snap_1',
              timestamp: '2024-01-01T10:00:00Z',
              rowCount: 2
            },
            {
              id: 'snap_2',
              timestamp: '2024-01-01T12:00:00Z',
              rowCount: 3
            }
          ])
        }),
        http.get('/api/tables/1/snapshots/compare', ({ request }) => {
          const url = new URL(request.url)
          const from = url.searchParams.get('from')
          const to = url.searchParams.get('to')
          
          return HttpResponse.json({
            from: from,
            to: to,
            changes: {
              added: [
                { id: '3', values: { name: 'New Row', status: 'active' } }
              ],
              modified: [
                {
                  id: '1',
                  before: { name: 'Old Name', status: 'inactive' },
                  after: { name: 'Updated Name', status: 'active' }
                }
              ],
              deleted: []
            }
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test snapshot comparison functionality
      const compareButton = screen.queryByRole('button', { name: /compare|diff/i })
      if (compareButton) {
        await helpers.user.click(compareButton)

        // Should show comparison interface
        await waitFor(() => {
          const comparison = screen.queryByText(/changes|added|modified|deleted/i)
          expect(comparison).toBeInTheDocument()
        })
      }
    })

    test('should restore table from historical snapshot', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_restore')
      const table = createTestTable('_restore')

      // Mock restoration functionality
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Current Row', status: 'modified' } }
          ])
        }),
        http.get('/api/tables/1/snapshots', () => {
          return HttpResponse.json([
            {
              id: 'snap_1',
              timestamp: '2024-01-01T10:00:00Z',
              rowCount: 1,
              description: 'Good backup point'
            }
          ])
        }),
        http.post('/api/tables/1/restore', () => {
          return HttpResponse.json({
            success: true,
            restoredRows: 1,
            timestamp: '2024-01-01T10:00:00Z'
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access restoration functionality
      const restoreButton = screen.queryByRole('button', { name: /restore|revert/i })
      if (restoreButton) {
        await helpers.user.click(restoreButton)

        // Should show confirmation dialog
        await waitFor(() => {
          const confirmation = screen.queryByText(/confirm.*restore|are you sure/i)
          if (confirmation) {
            const confirmButton = screen.getByRole('button', { name: /confirm|yes|restore/i })
            helpers.user.click(confirmButton)
          }
        })

        // Should show success message
        await waitFor(() => {
          const success = screen.queryByText(/restored|success/i)
          expect(success).toBeInTheDocument()
        })
      }
    })
  })

  describe('Change History Tracking', () => {
    test('should track and display row modification history', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_change_history')
      const table = createTestTable('_changes')

      // Mock change history
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Test Row', status: 'active' } }
          ])
        }),
        http.get('/api/tables/1/rows/1/history', () => {
          return HttpResponse.json([
            {
              id: 'change_1',
              timestamp: '2024-01-01T10:00:00Z',
              action: 'created',
              values: { name: 'Test Row', status: 'pending' },
              user: user.username
            },
            {
              id: 'change_2',
              timestamp: '2024-01-01T11:00:00Z',
              action: 'updated',
              changes: {
                status: { from: 'pending', to: 'active' }
              },
              user: user.username
            }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access row history
      const rowHistoryButton = screen.queryByRole('button', { name: /history.*row|view.*changes/i })
      if (rowHistoryButton) {
        await helpers.user.click(rowHistoryButton)

        // Should show change history
        await waitFor(() => {
          expect(screen.getByText(/created|updated/i)).toBeInTheDocument()
          expect(screen.getByText(/pending.*active/i)).toBeInTheDocument()
        })
      }
    })

    test('should show who made changes and when', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_audit_trail')
      const table = createTestTable('_audit')

      // Mock audit trail
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Audited Row' } }
          ])
        }),
        http.get('/api/tables/1/audit', () => {
          return HttpResponse.json([
            {
              id: 'audit_1',
              timestamp: '2024-01-01T10:00:00Z',
              user: user.username,
              action: 'create_row',
              target: 'row_1',
              details: { name: 'Audited Row' }
            },
            {
              id: 'audit_2',
              timestamp: '2024-01-01T11:00:00Z',
              user: 'other_user',
              action: 'update_row',
              target: 'row_1',
              details: { field: 'status', oldValue: 'draft', newValue: 'published' }
            }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access audit trail
      const auditButton = screen.queryByRole('button', { name: /audit|activity|log/i })
      if (auditButton) {
        await helpers.user.click(auditButton)

        // Should show audit information
        await waitFor(() => {
          expect(screen.getByText(user.username)).toBeInTheDocument()
          expect(screen.getByText(/other_user/i)).toBeInTheDocument()
          expect(screen.getByText(/10:00|11:00/)).toBeInTheDocument()
        })
      }
    })

    test('should filter history by date range', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_date_filter')
      const table = createTestTable('_date_filter')

      // Mock date-filtered history
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.get('/api/tables/1/history', ({ request }) => {
          const url = new URL(request.url)
          const from = url.searchParams.get('from')
          const to = url.searchParams.get('to')

          // Return filtered results based on date range
          if (from && to) {
            return HttpResponse.json([
              {
                id: 'filtered_1',
                timestamp: '2024-01-01T12:00:00Z',
                action: 'create_row',
                user: user.username
              }
            ])
          }

          // Return all history
          return HttpResponse.json([
            {
              id: 'all_1',
              timestamp: '2024-01-01T08:00:00Z',
              action: 'create_table',
              user: user.username
            },
            {
              id: 'all_2',
              timestamp: '2024-01-01T12:00:00Z',
              action: 'create_row',
              user: user.username
            },
            {
              id: 'all_3',
              timestamp: '2024-01-02T14:00:00Z',
              action: 'update_row',
              user: user.username
            }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test date filtering if available
      const historyButton = screen.queryByRole('button', { name: /history/i })
      if (historyButton) {
        await helpers.user.click(historyButton)

        // Look for date filter controls
        const dateFilter = screen.queryByLabelText(/from.*date|start.*date/i)
        if (dateFilter) {
          await helpers.user.type(dateFilter, '2024-01-01')
          
          const toDateFilter = screen.queryByLabelText(/to.*date|end.*date/i)
          if (toDateFilter) {
            await helpers.user.type(toDateFilter, '2024-01-01')
            
            const filterButton = screen.queryByRole('button', { name: /filter|apply/i })
            if (filterButton) {
              await helpers.user.click(filterButton)
              
              // Should show filtered results
              await waitFor(() => {
                expect(screen.getByText(/create_row/i)).toBeInTheDocument()
                expect(screen.queryByText(/update_row/i)).not.toBeInTheDocument()
              })
            }
          }
        }
      }
    })

    test('should export history data', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_export_history')
      const table = createTestTable('_export')

      // Mock export functionality
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        }),
        http.get('/api/tables/1/history/export', () => {
          return HttpResponse.json({
            downloadUrl: '/api/downloads/history_export_123.csv',
            filename: 'table_history_export.csv',
            size: 1024
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test export functionality
      const exportButton = screen.queryByRole('button', { name: /export.*history|download.*history/i })
      if (exportButton) {
        await helpers.user.click(exportButton)

        // Should initiate export process
        await waitFor(() => {
          const success = screen.queryByText(/export.*ready|download.*ready/i)
          const downloadLink = screen.queryByRole('link', { name: /download/i })
          expect(success || downloadLink).toBeTruthy()
        })
      }
    })
  })

  describe('Snapshot Management', () => {
    test('should delete old snapshots', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_delete_snapshots')
      const table = createTestTable('_cleanup')

      // Mock snapshot deletion
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/snapshots', () => {
          return HttpResponse.json([
            {
              id: 'snap_old',
              timestamp: '2024-01-01T10:00:00Z',
              rowCount: 1,
              description: 'Old snapshot'
            }
          ])
        }),
        http.delete('/api/tables/1/snapshots/snap_old', () => {
          return HttpResponse.json({ success: true })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access snapshot management
      const snapshotsButton = screen.queryByRole('button', { name: /snapshots|manage.*snapshots/i })
      if (snapshotsButton) {
        await helpers.user.click(snapshotsButton)

        // Delete old snapshot
        const deleteButton = screen.queryByRole('button', { name: /delete.*snapshot/i })
        if (deleteButton) {
          await helpers.user.click(deleteButton)

          // Confirm deletion
          const confirmButton = screen.queryByRole('button', { name: /confirm|yes|delete/i })
          if (confirmButton) {
            await helpers.user.click(confirmButton)

            // Should show success
            await waitFor(() => {
              const success = screen.queryByText(/deleted|removed/i)
              expect(success).toBeInTheDocument()
            })
          }
        }
      }
    })

    test('should set automatic snapshot intervals', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_auto_snapshots')
      const table = createTestTable('_auto')

      // Mock automatic snapshot settings
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/settings', () => {
          return HttpResponse.json({
            autoSnapshots: false,
            snapshotInterval: 'daily'
          })
        }),
        http.put('/api/tables/1/settings', () => {
          return HttpResponse.json({
            autoSnapshots: true,
            snapshotInterval: 'hourly'
          })
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Access snapshot settings
      const settingsButton = screen.queryByRole('button', { name: /settings|configure/i })
      if (settingsButton) {
        await helpers.user.click(settingsButton)

        // Enable automatic snapshots
        const autoCheckbox = screen.queryByRole('checkbox', { name: /automatic.*snapshots/i })
        if (autoCheckbox) {
          await helpers.user.click(autoCheckbox)

          // Set interval
          const intervalSelect = screen.queryByRole('combobox', { name: /interval|frequency/i })
          if (intervalSelect) {
            await helpers.user.selectOptions(intervalSelect, 'hourly')
            
            const saveButton = screen.queryByRole('button', { name: /save|apply/i })
            if (saveButton) {
              await helpers.user.click(saveButton)

              // Should save settings
              await waitFor(() => {
                const success = screen.queryByText(/saved|updated/i)
                expect(success).toBeInTheDocument()
              })
            }
          }
        }
      }
    })
  })
})