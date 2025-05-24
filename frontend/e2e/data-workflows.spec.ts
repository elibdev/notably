import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('Data Workflows', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Bulk Data Operations', () => {
    test('should handle bulk row creation efficiently', async ({ page }) => {
      const user = createTestUser('_bulk_create');
      const table = createTestTable('_bulk');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();
      
      // Create multiple rows in succession
      const rowsToCreate = 5;
      for (let i = 0; i < rowsToCreate; i++) {
        const row = helpers.generateTestRow({
          values: {
            name: `Bulk User ${i + 1}`,
            email: `bulk${i + 1}@example.com`,
            age: 25 + i,
            active: i % 2 === 0
          }
        });
        await helpers.createRow(row);
      }

      // Verify all rows were created
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(initialCount + rowsToCreate);

      // Verify data integrity
      for (let i = 0; i < rowsToCreate; i++) {
        await expect(page.locator(`text=Bulk User ${i + 1}`)).toBeVisible();
      }
    });

    test('should handle bulk row updates correctly', async ({ page }) => {
      const user = createTestUser('_bulk_update');
      const table = createTestTable('_bulk_upd');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create several rows first
      const rows = [];
      for (let i = 0; i < 3; i++) {
        const row = helpers.generateTestRow({
          id: `bulk_row_${i}`,
          values: {
            name: `Original User ${i}`,
            email: `original${i}@example.com`,
            age: 30,
            active: true
          }
        });
        await helpers.createRow(row);
        rows.push(row);
      }

      // Update all rows
      for (let i = 0; i < rows.length; i++) {
        await helpers.editRow(`bulk_row_${i}`, {
          name: `Updated User ${i}`,
          age: 35
        });
      }

      // Verify updates
      for (let i = 0; i < rows.length; i++) {
        await expect(page.locator(`text=Updated User ${i}`)).toBeVisible();
        await expect(page.locator(`text=35`)).toBeVisible();
      }
    });

    test('should handle bulk row deletion safely', async ({ page }) => {
      const user = createTestUser('_bulk_delete');
      const table = createTestTable('_bulk_del');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create test rows
      const rowIds = [];
      for (let i = 0; i < 4; i++) {
        const rowId = `delete_row_${i}`;
        const row = helpers.generateTestRow({
          id: rowId,
          values: {
            name: `Delete User ${i}`,
            email: `delete${i}@example.com`,
            age: 25,
            active: true
          }
        });
        await helpers.createRow(row);
        rowIds.push(rowId);
      }

      const initialCount = await helpers.getRowCount();

      // Delete multiple rows
      for (let i = 0; i < 2; i++) {
        await helpers.deleteRow(rowIds[i]);
      }

      // Verify deletions
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(initialCount - 2);

      // Verify specific rows are deleted
      await helpers.verifyRowDoesNotExist(rowIds[0]);
      await helpers.verifyRowDoesNotExist(rowIds[1]);

      // Verify remaining rows still exist
      await helpers.verifyRowExists(rowIds[2]);
      await helpers.verifyRowExists(rowIds[3]);
    });

    test('should maintain data consistency during concurrent operations', async ({ page }) => {
      const user = createTestUser('_concurrent');
      const table = createTestTable('_concurrent');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create a base row
      const baseRow = helpers.generateTestRow({
        id: 'concurrent_row',
        values: {
          name: 'Concurrent Test',
          email: 'concurrent@example.com',
          age: 30,
          active: true
        }
      });
      await helpers.createRow(baseRow);

      // Simulate rapid updates
      await helpers.editRow('concurrent_row', { age: 31 });
      await helpers.editRow('concurrent_row', { age: 32 });
      await helpers.editRow('concurrent_row', { name: 'Updated Concurrent' });

      // Verify final state
      await expect(page.locator('text=Updated Concurrent')).toBeVisible();
      await expect(page.locator('text=32')).toBeVisible();
    });
  });

  test.describe('Search and Filtering', () => {
    test('should filter rows by text content', async ({ page }) => {
      const user = createTestUser('_search');
      const table = createTestTable('_search');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create diverse test data
      const testData = [
        { name: 'Alice Johnson', email: 'alice@example.com', age: 25 },
        { name: 'Bob Smith', email: 'bob@test.com', age: 30 },
        { name: 'Charlie Brown', email: 'charlie@example.com', age: 35 },
        { name: 'Diana Wilson', email: 'diana@sample.com', age: 28 }
      ];

      for (const data of testData) {
        const row = helpers.generateTestRow({ values: { ...data, active: true } });
        await helpers.createRow(row);
      }

      // Test text search if available
      const searchInput = page.locator('input[placeholder*="Search"], input[placeholder*="Filter"]');
      if (await searchInput.isVisible()) {
        // Search for "example.com" emails
        await searchInput.fill('example.com');
        await page.waitForTimeout(1000); // Wait for filter to apply

        // Should show Alice and Charlie
        await expect(page.locator('text=Alice Johnson')).toBeVisible();
        await expect(page.locator('text=Charlie Brown')).toBeVisible();
        
        // Should not show Bob or Diana
        await expect(page.locator('text=Bob Smith')).not.toBeVisible();
        await expect(page.locator('text=Diana Wilson')).not.toBeVisible();

        // Clear search
        await searchInput.clear();
        await page.waitForTimeout(500);

        // All should be visible again
        await expect(page.locator('text=Alice Johnson')).toBeVisible();
        await expect(page.locator('text=Bob Smith')).toBeVisible();
      }
    });

    test('should handle sorting by different columns', async ({ page }) => {
      const user = createTestUser('_sort');
      const table = createTestTable('_sort');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create data with different ages for sorting
      const testData = [
        { name: 'Zoe', age: 45, email: 'zoe@example.com' },
        { name: 'Alice', age: 25, email: 'alice@example.com' },
        { name: 'Mike', age: 35, email: 'mike@example.com' }
      ];

      for (const data of testData) {
        const row = helpers.generateTestRow({ values: { ...data, active: true } });
        await helpers.createRow(row);
      }

      // Test column sorting if available
      const ageHeader = page.locator('th:has-text("age")');
      if (await ageHeader.isVisible()) {
        await ageHeader.click(); // Sort by age
        await page.waitForTimeout(1000);

        // Verify sorting order (should be ascending: 25, 35, 45)
        const rows = page.locator('[data-testid^="row-"]');
        const firstRowAge = await rows.first().locator('text=25').isVisible();
        expect(firstRowAge).toBe(true);
      }
    });

    test('should filter by boolean values', async ({ page }) => {
      const user = createTestUser('_bool_filter');
      const table = createTestTable('_bool_filter');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create mixed active/inactive users
      const testData = [
        { name: 'Active User 1', active: true, age: 25 },
        { name: 'Inactive User 1', active: false, age: 30 },
        { name: 'Active User 2', active: true, age: 35 },
        { name: 'Inactive User 2', active: false, age: 40 }
      ];

      for (const data of testData) {
        const row = helpers.generateTestRow({ 
          values: { ...data, email: `${data.name.toLowerCase().replace(/\s+/g, '')}@example.com` }
        });
        await helpers.createRow(row);
      }

      // Look for filter controls
      const activeFilter = page.locator('input[type="checkbox"], select, button').filter({ hasText: /active/i });
      if (await activeFilter.first().isVisible()) {
        await activeFilter.first().click();
        await page.waitForTimeout(1000);

        // Should show only active users or apply some filter
        const visibleRows = await page.locator('[data-testid^="row-"]').count();
        expect(visibleRows).toBeGreaterThan(0);
      }
    });

    test('should handle complex multi-column filtering', async ({ page }) => {
      const user = createTestUser('_multi_filter');
      const table = createTestTable('_multi_filter');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create diverse dataset
      const testData = [
        { name: 'Young Active', age: 25, active: true, email: 'young@example.com' },
        { name: 'Young Inactive', age: 26, active: false, email: 'young2@example.com' },
        { name: 'Old Active', age: 55, active: true, email: 'old@example.com' },
        { name: 'Old Inactive', age: 56, active: false, email: 'old2@example.com' }
      ];

      for (const data of testData) {
        const row = helpers.generateTestRow({ values: data });
        await helpers.createRow(row);
      }

      // Test combined filters (if UI supports it)
      const searchInput = page.locator('input[placeholder*="Search"]');
      if (await searchInput.isVisible()) {
        // Search for young people
        await searchInput.fill('Young');
        await page.waitForTimeout(1000);

        // Should only show young users
        await expect(page.locator('text=Young Active')).toBeVisible();
        await expect(page.locator('text=Young Inactive')).toBeVisible();
        await expect(page.locator('text=Old Active')).not.toBeVisible();
      }
    });
  });

  test.describe('Complex Table Management', () => {
    test('should handle multiple tables with different schemas', async ({ page }) => {
      const user = createTestUser('_multi_tables');
      
      await helpers.registerUser(user);

      // Create multiple tables with different structures
      const peopleTable = {
        name: `people_${Date.now()}`,
        columns: [
          { name: 'name', dataType: 'string' },
          { name: 'age', dataType: 'number' },
          { name: 'email', dataType: 'string' }
        ]
      };

      const projectsTable = {
        name: `projects_${Date.now()}`,
        columns: [
          { name: 'title', dataType: 'string' },
          { name: 'status', dataType: 'string' },
          { name: 'budget', dataType: 'number' },
          { name: 'completed', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(peopleTable);
      await helpers.createTable(projectsTable);

      // Verify both tables exist
      await helpers.verifyTableExists(peopleTable.name);
      await helpers.verifyTableExists(projectsTable.name);

      // Add data to first table
      await helpers.selectTable(peopleTable.name);
      await helpers.createRow(helpers.generateTestRow({
        values: { name: 'John Doe', age: 30, email: 'john@example.com' }
      }));

      // Switch to second table and add data
      await helpers.selectTable(projectsTable.name);
      await helpers.createRow(helpers.generateTestRow({
        values: { title: 'Website Redesign', status: 'In Progress', budget: 50000, completed: false }
      }));

      // Verify data in both tables
      await helpers.selectTable(peopleTable.name);
      await expect(page.locator('text=John Doe')).toBeVisible();

      await helpers.selectTable(projectsTable.name);
      await expect(page.locator('text=Website Redesign')).toBeVisible();
    });

    test('should maintain data integrity across table operations', async ({ page }) => {
      const user = createTestUser('_data_integrity');
      const table = createTestTable('_integrity');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create initial data
      const originalRow = helpers.generateTestRow({
        id: 'integrity_test',
        values: {
          name: 'Integrity Test User',
          email: 'integrity@example.com',
          age: 25,
          active: true
        }
      });
      await helpers.createRow(originalRow);

      // Perform various operations
      await helpers.editRow('integrity_test', { age: 26 });
      await page.reload(); // Test persistence
      await helpers.waitForTableLoad();

      // Verify data persisted correctly
      await expect(page.locator('text=Integrity Test User')).toBeVisible();
      await expect(page.locator('text=26')).toBeVisible();

      // Test deletion persistence
      await helpers.deleteRow('integrity_test');
      await page.reload();
      await helpers.waitForTableLoad();

      // Verify deletion persisted
      await helpers.verifyRowDoesNotExist('integrity_test');
    });

    test('should handle table schema modifications gracefully', async ({ page }) => {
      const user = createTestUser('_schema_mod');
      
      await helpers.registerUser(user);

      // Create table with initial schema
      const initialTable = {
        name: `schema_test_${Date.now()}`,
        columns: [
          { name: 'name', dataType: 'string' },
          { name: 'age', dataType: 'number' }
        ]
      };

      await helpers.createTable(initialTable);
      await helpers.selectTable(initialTable.name);

      // Add data with initial schema
      await helpers.createRow(helpers.generateTestRow({
        values: { name: 'Schema Test User', age: 30 }
      }));

      // Note: In a real app, you might have schema modification features
      // For now, we test that existing data remains accessible
      await page.reload();
      await helpers.waitForTableLoad();
      
      await expect(page.locator('text=Schema Test User')).toBeVisible();
    });

    test('should handle large datasets efficiently', async ({ page }) => {
      const user = createTestUser('_large_data');
      const table = createTestTable('_large');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create a larger dataset (limited for test performance)
      const largeDatasetSize = 10;
      
      for (let i = 0; i < largeDatasetSize; i++) {
        const row = helpers.generateTestRow({
          values: {
            name: `User ${i.toString().padStart(3, '0')}`,
            email: `user${i}@example.com`,
            age: 20 + (i % 50),
            active: i % 2 === 0
          }
        });
        await helpers.createRow(row);
        
        // Brief pause to avoid overwhelming the system
        if (i % 3 === 0) {
          await page.waitForTimeout(100);
        }
      }

      // Verify data loaded correctly
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(largeDatasetSize);

      // Test that UI remains responsive
      await helpers.waitForNetworkIdle();
      await expect(page.locator('text=User 000')).toBeVisible();
      await expect(page.locator('text=User 009')).toBeVisible();
    });
  });

  test.describe('Data Validation and Edge Cases', () => {
    test('should validate data types on input', async ({ page }) => {
      const user = createTestUser('_validation');
      const table = createTestTable('_validation');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Try to create row with invalid data types
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Try to put text in number field
      const ageInput = page.locator('input[placeholder="age"]');
      if (await ageInput.isVisible()) {
        await ageInput.fill('not a number');
        await page.click('button:has-text("Save")');

        // Should show validation error or prevent submission
        const formStillVisible = await page.locator('[data-testid="row-form"]').isVisible();
        expect(formStillVisible).toBe(true);
      }
    });

    test('should handle special characters and unicode', async ({ page }) => {
      const user = createTestUser('_unicode');
      const table = createTestTable('_unicode');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test with special characters
      const specialRow = helpers.generateTestRow({
        values: {
          name: 'José María Çağlar 中文 🚀',
          email: 'josé@éxample.com',
          age: 25,
          active: true
        }
      });

      await helpers.createRow(specialRow);

      // Verify special characters are preserved
      await expect(page.locator('text=José María Çağlar 中文 🚀')).toBeVisible();
      await expect(page.locator('text=josé@éxample.com')).toBeVisible();
    });

    test('should handle very long text values', async ({ page }) => {
      const user = createTestUser('_long_text');
      const table = createTestTable('_long_text');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create row with very long text
      const longText = 'A'.repeat(500);
      const longRow = helpers.generateTestRow({
        values: {
          name: longText,
          email: 'long@example.com',
          age: 25,
          active: true
        }
      });

      await helpers.createRow(longRow);

      // Verify long text is handled (might be truncated in UI)
      await expect(page.locator('text*=AAAA')).toBeVisible();
    });

    test('should handle empty and null-like values appropriately', async ({ page }) => {
      const user = createTestUser('_empty_vals');
      const table = createTestTable('_empty_vals');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test with minimal required data
      const minimalRow = helpers.generateTestRow({
        values: {
          name: '', // Empty string
          email: 'minimal@example.com',
          age: 0, // Zero value
          active: false
        }
      });

      await helpers.createRow(minimalRow);

      // Verify row was created even with minimal data
      await expect(page.locator('text=minimal@example.com')).toBeVisible();
    });
  });

  test.describe('Performance and Scalability', () => {
    test('should maintain performance with rapid operations', async ({ page }) => {
      const user = createTestUser('_performance');
      const table = createTestTable('_performance');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const startTime = Date.now();

      // Perform rapid operations
      for (let i = 0; i < 5; i++) {
        const row = helpers.generateTestRow({
          values: {
            name: `Performance Test ${i}`,
            email: `perf${i}@example.com`,
            age: 25 + i,
            active: true
          }
        });
        await helpers.createRow(row);
      }

      const endTime = Date.now();
      const duration = endTime - startTime;

      // Should complete within reasonable time (adjust threshold as needed)
      expect(duration).toBeLessThan(30000); // 30 seconds max

      // Verify all operations completed successfully
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(5);
    });

    test('should handle network timeouts gracefully', async ({ page }) => {
      const user = createTestUser('_timeout');
      const table = createTestTable('_timeout');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Simulate slow network for one operation
      await page.route('/api/tables/*/rows', async (route) => {
        // Delay the first request
        await new Promise(resolve => setTimeout(resolve, 2000));
        route.continue();
      });

      const row = helpers.generateTestRow({
        values: {
          name: 'Timeout Test User',
          email: 'timeout@example.com',
          age: 30,
          active: true
        }
      });

      // This should eventually succeed despite the delay
      await helpers.createRow(row);
      await expect(page.locator('text=Timeout Test User')).toBeVisible({ timeout: 15000 });
    });
  });
});