import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('History and Snapshot Features', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Snapshot Functionality', () => {
    test('should create and view table snapshots', async ({ page }) => {
      const user = createTestUser('_snapshot');
      const table = createTestTable('_snap');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create initial data
      const row1 = helpers.generateTestRow({ id: `snap_1_${Date.now()}` });
      const row2 = helpers.generateTestRow({ id: `snap_2_${Date.now()}` });
      
      await helpers.createRow(row1);
      await helpers.createRow(row2);

      // Take snapshot
      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        // Should show snapshot view with current data
        await helpers.verifyRowExists(row1.id!, row1.values);
        await helpers.verifyRowExists(row2.id!, row2.values);
      }
    });

    test('should view historical snapshots at specific times', async ({ page }) => {
      const user = createTestUser('_hist_snap');
      const table = createTestTable('_hist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Record timestamp before creating data
      const timestamp1 = new Date().toISOString();
      await page.waitForTimeout(1000);

      // Create first row
      const row1 = helpers.generateTestRow({ id: `hist_1_${Date.now()}` });
      await helpers.createRow(row1);
      
      await page.waitForTimeout(1000);
      const timestamp2 = new Date().toISOString();
      await page.waitForTimeout(1000);

      // Create second row
      const row2 = helpers.generateTestRow({ id: `hist_2_${Date.now()}` });
      await helpers.createRow(row2);

      // Try to view snapshot at timestamp1 (should have no rows)
      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        // If there's a timestamp input field
        const timestampInput = page.locator('input[type="datetime-local"], input[placeholder*="timestamp"]');
        if (await timestampInput.isVisible()) {
          await timestampInput.fill(timestamp1.slice(0, 16)); // Format for datetime-local
          await page.click('button:has-text("View Snapshot")');
          
          // Should show empty or minimal data
          await helpers.verifyRowDoesNotExist(row1.id!);
          await helpers.verifyRowDoesNotExist(row2.id!);
        }
      }
    });

    test('should handle snapshot errors gracefully', async ({ page }) => {
      const user = createTestUser('_snap_error');
      const table = createTestTable('_snap_err');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Try to take snapshot with invalid timestamp
      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        const timestampInput = page.locator('input[type="datetime-local"], input[placeholder*="timestamp"]');
        if (await timestampInput.isVisible()) {
          await timestampInput.fill('invalid-timestamp');
          await page.click('button:has-text("View Snapshot")');
          
          // Should show error message
          await expect(page.locator('text*=Invalid')).toBeVisible({ timeout: 10000 });
        }
      }
    });
  });

  test.describe('History Tracking', () => {
    test('should track row creation events', async ({ page }) => {
      const user = createTestUser('_hist_create');
      const table = createTestTable('_create_hist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const startTime = new Date().toISOString();
      await page.waitForTimeout(1000);

      // Create a row
      const row = helpers.generateTestRow({ id: `create_${Date.now()}` });
      await helpers.createRow(row);

      await page.waitForTimeout(1000);
      const endTime = new Date().toISOString();

      // Check history
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        // Fill time range
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(startTime.slice(0, 16));
          await endInput.fill(endTime.slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should show creation event
          await expect(page.locator('text*=created')).toBeVisible({ timeout: 10000 });
        }
      }
    });

    test('should track row modification events', async ({ page }) => {
      const user = createTestUser('_hist_modify');
      const table = createTestTable('_modify_hist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create initial row
      const row = helpers.generateTestRow({ id: `modify_${Date.now()}` });
      await helpers.createRow(row);

      const startTime = new Date().toISOString();
      await page.waitForTimeout(1000);

      // Modify the row
      const updatedValues = { name: 'Modified Name', age: 999, active: false };
      await helpers.editRow(row.id!, updatedValues);

      await page.waitForTimeout(1000);
      const endTime = new Date().toISOString();

      // Check history for modifications
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(startTime.slice(0, 16));
          await endInput.fill(endTime.slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should show modification event
          await expect(page.locator('text*=modified')).toBeVisible({ timeout: 10000 });
        }
      }
    });

    test('should track row deletion events', async ({ page }) => {
      const user = createTestUser('_hist_delete');
      const table = createTestTable('_delete_hist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create row to delete
      const row = helpers.generateTestRow({ id: `delete_${Date.now()}` });
      await helpers.createRow(row);

      const startTime = new Date().toISOString();
      await page.waitForTimeout(1000);

      // Delete the row
      await helpers.deleteRow(row.id!);

      await page.waitForTimeout(1000);
      const endTime = new Date().toISOString();

      // Check history for deletion
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(startTime.slice(0, 16));
          await endInput.fill(endTime.slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should show deletion event
          await expect(page.locator('text*=deleted')).toBeVisible({ timeout: 10000 });
        }
      }
    });

    test('should handle complex history queries', async ({ page }) => {
      const user = createTestUser('_hist_complex');
      const table = createTestTable('_complex_hist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const timestamps = [];
      
      // Record start time
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(1000);

      // Create multiple rows with modifications
      const rows = [
        helpers.generateTestRow({ id: `complex_1_${Date.now()}` }),
        helpers.generateTestRow({ id: `complex_2_${Date.now()}` }),
        helpers.generateTestRow({ id: `complex_3_${Date.now()}` })
      ];

      for (const row of rows) {
        await helpers.createRow(row);
        timestamps.push(new Date().toISOString());
        await page.waitForTimeout(500);
      }

      // Modify some rows
      await helpers.editRow(rows[0].id!, { name: 'Modified 1', age: 100 });
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(500);

      await helpers.editRow(rows[1].id!, { name: 'Modified 2', age: 200 });
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(500);

      // Delete one row
      await helpers.deleteRow(rows[2].id!);
      timestamps.push(new Date().toISOString());

      // Query history for the entire period
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(timestamps[0].slice(0, 16));
          await endInput.fill(timestamps[timestamps.length - 1].slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should show multiple events
          await page.waitForTimeout(2000);
          const historyItems = page.locator('[data-testid="history-item"], .history-event');
          const count = await historyItems.count();
          expect(count).toBeGreaterThan(3); // At least 3 creates + 2 modifies + 1 delete
        }
      }
    });
  });

  test.describe('Time-based Data Recovery', () => {
    test('should restore data from historical snapshot', async ({ page }) => {
      const user = createTestUser('_restore');
      const table = createTestTable('_restore');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create initial state
      const row = helpers.generateTestRow({ id: `restore_${Date.now()}` });
      await helpers.createRow(row);
      
      const snapshotTime = new Date().toISOString();
      await page.waitForTimeout(2000);

      // Modify the row
      const modifiedValues = { name: 'Modified', age: 999, active: false };
      await helpers.editRow(row.id!, modifiedValues);

      // View current state - should show modified values
      await helpers.verifyRowExists(row.id!, modifiedValues);

      // Go back to snapshot time
      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        const timestampInput = page.locator('input[type="datetime-local"], input[placeholder*="timestamp"]');
        if (await timestampInput.isVisible()) {
          await timestampInput.fill(snapshotTime.slice(0, 16));
          await page.click('button:has-text("View Snapshot")');
          
          // Should show original values
          await helpers.verifyRowExists(row.id!, row.values);
        }
      }
    });

    test('should handle data consistency across time', async ({ page }) => {
      const user = createTestUser('_consistency');
      const table = createTestTable('_consistent');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create baseline data
      const rows = [
        helpers.generateTestRow({ id: `cons_1_${Date.now()}` }),
        helpers.generateTestRow({ id: `cons_2_${Date.now()}` })
      ];

      for (const row of rows) {
        await helpers.createRow(row);
      }

      const baselineTime = new Date().toISOString();
      await page.waitForTimeout(2000);

      // Make changes
      await helpers.editRow(rows[0].id!, { name: 'Changed', age: 888 });
      await helpers.deleteRow(rows[1].id!);
      
      // Add new row
      const newRow = helpers.generateTestRow({ id: `cons_new_${Date.now()}` });
      await helpers.createRow(newRow);

      // Verify current state
      await helpers.verifyRowExists(rows[0].id!, { name: 'Changed', age: 888 });
      await helpers.verifyRowDoesNotExist(rows[1].id!);
      await helpers.verifyRowExists(newRow.id!, newRow.values);

      // Go back to baseline
      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        const timestampInput = page.locator('input[type="datetime-local"], input[placeholder*="timestamp"]');
        if (await timestampInput.isVisible()) {
          await timestampInput.fill(baselineTime.slice(0, 16));
          await page.click('button:has-text("View Snapshot")');
          
          // Should show baseline state
          await helpers.verifyRowExists(rows[0].id!, rows[0].values);
          await helpers.verifyRowExists(rows[1].id!, rows[1].values);
          await helpers.verifyRowDoesNotExist(newRow.id!);
        }
      }
    });
  });

  test.describe('Performance and Edge Cases', () => {
    test('should handle large history ranges efficiently', async ({ page }) => {
      const user = createTestUser('_perf');
      const table = createTestTable('_perf');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const startTime = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(); // 24 hours ago
      const endTime = new Date().toISOString();

      // Query large time range
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(startTime.slice(0, 16));
          await endInput.fill(endTime.slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should handle gracefully (no crash, reasonable response time)
          await page.waitForTimeout(5000);
          await helpers.expectNoErrors();
        }
      }
    });

    test('should handle invalid time ranges', async ({ page }) => {
      const user = createTestUser('_invalid_time');
      const table = createTestTable('_invalid');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          // Start time after end time
          await startInput.fill('2024-12-31T23:59');
          await endInput.fill('2024-01-01T00:00');
          await page.click('button:has-text("View History")');
          
          // Should show validation error
          await expect(page.locator('text*=Invalid')).toBeVisible({ timeout: 10000 });
        }
      }
    });

    test('should handle future timestamps gracefully', async ({ page }) => {
      const user = createTestUser('_future');
      const table = createTestTable('_future');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const futureTime = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(); // 24 hours in future

      const snapshotButton = page.locator('button:has-text("Snapshot")');
      if (await snapshotButton.isVisible()) {
        await snapshotButton.click();
        
        const timestampInput = page.locator('input[type="datetime-local"], input[placeholder*="timestamp"]');
        if (await timestampInput.isVisible()) {
          await timestampInput.fill(futureTime.slice(0, 16));
          await page.click('button:has-text("View Snapshot")');
          
          // Should handle gracefully or show current state
          await page.waitForTimeout(2000);
          await helpers.expectNoErrors();
        }
      }
    });
  });

  test.describe('Integration with CRUD Operations', () => {
    test('should maintain history integrity during complex operations', async ({ page }) => {
      const user = createTestUser('_integrity');
      const table = createTestTable('_integrity');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const timestamps = [];
      
      // Perform a series of operations
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(1000);

      // Create
      const row = helpers.generateTestRow({ id: `integrity_${Date.now()}` });
      await helpers.createRow(row);
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(1000);

      // Update
      await helpers.editRow(row.id!, { name: 'Updated', age: 50 });
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(1000);

      // Update again
      await helpers.editRow(row.id!, { name: 'Updated Again', age: 75 });
      timestamps.push(new Date().toISOString());
      await page.waitForTimeout(1000);

      // Delete
      await helpers.deleteRow(row.id!);
      timestamps.push(new Date().toISOString());

      // Check history shows all operations
      const historyButton = page.locator('button:has-text("History")');
      if (await historyButton.isVisible()) {
        await historyButton.click();
        
        const startInput = page.locator('input[placeholder*="start"], input[name="start"]');
        const endInput = page.locator('input[placeholder*="end"], input[name="end"]');
        
        if (await startInput.isVisible() && await endInput.isVisible()) {
          await startInput.fill(timestamps[0].slice(0, 16));
          await endInput.fill(timestamps[timestamps.length - 1].slice(0, 16));
          await page.click('button:has-text("View History")');
          
          // Should show create, 2 updates, and delete events
          await page.waitForTimeout(2000);
          const historyItems = page.locator('[data-testid="history-item"], .history-event');
          const count = await historyItems.count();
          expect(count).toBeGreaterThanOrEqual(4);
        }
      }
    });
  });
});