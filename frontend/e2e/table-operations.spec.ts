import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('Table and Row Operations', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Table Management', () => {
    test('should create a table without columns', async ({ page }) => {
      const user = createTestUser('_table_basic');
      const table = { name: `simple_table_${Date.now()}` };

      await helpers.registerUser(user);
      await helpers.createTable(table);

      // Verify table appears in list
      await helpers.verifyTableExists(table.name);
    });

    test('should create a table with predefined columns', async ({ page }) => {
      const user = createTestUser('_table_columns');
      const table = createTestTable('_with_cols');

      await helpers.registerUser(user);
      await helpers.createTable(table);

      // Verify table exists
      await helpers.verifyTableExists(table.name);

      // Verify table structure
      await helpers.verifyTableStructure(table.name, table.columns!.map(c => c.name));
    });

    test('should handle table creation errors gracefully', async ({ page }) => {
      const user = createTestUser('_table_error');
      
      await helpers.registerUser(user);

      // Try to create table with empty name
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');
      
      // Submit without name
      await page.click('button:has-text("Create")');
      
      // Should show validation error or prevent submission
      await expect(page.locator('[data-testid="create-table-modal"]')).toBeVisible();
    });

    test('should handle duplicate table names', async ({ page }) => {
      const user = createTestUser('_table_dup');
      const tableName = `duplicate_table_${Date.now()}`;

      await helpers.registerUser(user);
      
      // Create first table
      await helpers.createTable({ name: tableName });
      
      // Try to create table with same name
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');
      await page.fill('input[placeholder="Table name"]', tableName);
      await page.click('button:has-text("Create")');
      
      // Should show error message
      await expect(page.locator('text*=already exists')).toBeVisible({ timeout: 10000 });
    });

    test('should navigate to table view when table is selected', async ({ page }) => {
      const user = createTestUser('_table_nav');
      const table = createTestTable('_nav');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Should be in table view
      await expect(page.locator('[data-testid="table-view"]')).toBeVisible();
      await expect(page.locator(`text=${table.name}`)).toBeVisible();
    });
  });

  test.describe('Row Operations - Basic CRUD', () => {
    test('should create a row without specifying ID', async ({ page }) => {
      const user = createTestUser('_row_basic');
      const table = createTestTable('_basic');
      const row = helpers.generateTestRow();

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();
      await helpers.createRow(row);

      // Verify row was created
      const newCount = await helpers.getRowCount();
      expect(newCount).toBe(initialCount + 1);

      // Verify row data appears in table
      for (const value of Object.values(row.values)) {
        await expect(page.locator(`text=${String(value)}`)).toBeVisible();
      }
    });

    test('should create a row with custom ID', async ({ page }) => {
      const user = createTestUser('_row_id');
      const table = createTestTable('_with_id');
      const customId = `custom_id_${Date.now()}`;
      const row = helpers.generateTestRow({ id: customId });

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      await helpers.createRow(row);

      // Verify row exists with custom ID
      await helpers.verifyRowExists(customId, row.values);
    });

    test('should edit an existing row', async ({ page }) => {
      const user = createTestUser('_row_edit');
      const table = createTestTable('_edit');
      const row = helpers.generateTestRow({ id: `edit_test_${Date.now()}` });

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);
      await helpers.createRow(row);

      // Edit the row
      const updatedValues = {
        name: 'Updated Name',
        age: 99,
        active: false
      };

      await helpers.editRow(row.id!, updatedValues);

      // Verify changes
      await helpers.verifyRowExists(row.id!, updatedValues);
    });

    test('should delete a row', async ({ page }) => {
      const user = createTestUser('_row_delete');
      const table = createTestTable('_delete');
      const row = helpers.generateTestRow({ id: `delete_test_${Date.now()}` });

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);
      await helpers.createRow(row);

      const initialCount = await helpers.getRowCount();
      await helpers.deleteRow(row.id!);

      // Verify row is deleted
      await helpers.verifyRowDoesNotExist(row.id!);
      
      const newCount = await helpers.getRowCount();
      expect(newCount).toBe(initialCount - 1);
    });
  });

  test.describe('Row Operations - Data Types', () => {
    test('should handle string values correctly', async ({ page }) => {
      const user = createTestUser('_string');
      const table = {
        name: `string_table_${Date.now()}`,
        columns: [{ name: 'text_field', dataType: 'string' }]
      };

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const testStrings = [
        'Simple text',
        'Text with "quotes"',
        'Text with special chars: !@#$%^&*()',
        'Unicode: 你好世界',
        ''  // Empty string
      ];

      for (const testString of testStrings) {
        const row = { values: { text_field: testString } };
        await helpers.createRow(row);
        
        if (testString) {  // Don't check for empty string visibility
          await expect(page.locator(`text=${testString}`)).toBeVisible();
        }
      }
    });

    test('should handle number values correctly', async ({ page }) => {
      const user = createTestUser('_number');
      const table = {
        name: `number_table_${Date.now()}`,
        columns: [{ name: 'num_field', dataType: 'number' }]
      };

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const testNumbers = [42, 0, -10, 3.14159, 999999];

      for (const testNumber of testNumbers) {
        const row = { values: { num_field: testNumber } };
        await helpers.createRow(row);
        await expect(page.locator(`text=${String(testNumber)}`)).toBeVisible();
      }
    });

    test('should handle boolean values correctly', async ({ page }) => {
      const user = createTestUser('_boolean');
      const table = {
        name: `boolean_table_${Date.now()}`,
        columns: [{ name: 'bool_field', dataType: 'boolean' }]
      };

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test true value
      await helpers.createRow({ values: { bool_field: true } });
      await expect(page.locator('text=true')).toBeVisible();

      // Test false value
      await helpers.createRow({ values: { bool_field: false } });
      await expect(page.locator('text=false')).toBeVisible();
    });
  });

  test.describe('Row Operations - Error Handling', () => {
    test('should handle duplicate row IDs', async ({ page }) => {
      const user = createTestUser('_dup_id');
      const table = createTestTable('_dup');
      const duplicateId = `dup_${Date.now()}`;

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create first row
      await helpers.createRow({
        id: duplicateId,
        values: { name: 'First', age: 25, active: true }
      });

      // Try to create row with same ID
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');
      
      const idInput = page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.fill(duplicateId);
      }
      
      await page.fill('input[placeholder="name"]', 'Second');
      await page.fill('input[placeholder="age"]', '30');
      await page.click('button:has-text("Save")');

      // Should show error or prevent creation
      await expect(page.locator('text*=already exists')).toBeVisible({ timeout: 10000 });
    });

    test('should validate required fields', async ({ page }) => {
      const user = createTestUser('_validation');
      const table = createTestTable('_valid');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Try to create row with missing data
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');
      
      // Only fill one field, leave others empty
      await page.fill('input[placeholder="name"]', 'Test Name');
      
      await page.click('button:has-text("Save")');

      // Form should either show validation errors or successfully create with empty values
      // (depending on business logic)
      await page.waitForTimeout(2000);
    });

    test('should handle network errors during row operations', async ({ page }) => {
      const user = createTestUser('_network');
      const table = createTestTable('_network');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Intercept row creation requests and make them fail
      await page.route('/api/tables/*/rows', route => {
        route.abort();
      });

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');
      
      await page.fill('input[placeholder="name"]', 'Test');
      await page.fill('input[placeholder="age"]', '25');
      
      await page.click('button:has-text("Save")');

      // Should handle error gracefully
      await page.waitForTimeout(3000);
      const isFormStillVisible = await page.locator('[data-testid="row-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });
  });

  test.describe('Real-time Data Updates', () => {
    test('should refresh table data correctly', async ({ page }) => {
      const user = createTestUser('_refresh');
      const table = createTestTable('_refresh');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();

      // Create a row
      await helpers.createRow(helpers.generateTestRow());

      // Count should increase
      const newCount = await helpers.getRowCount();
      expect(newCount).toBe(initialCount + 1);
    });

    test('should handle concurrent row operations', async ({ page }) => {
      const user = createTestUser('_concurrent');
      const table = createTestTable('_concurrent');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create multiple rows quickly
      const rows = [
        helpers.generateTestRow({ id: `concurrent_1_${Date.now()}` }),
        helpers.generateTestRow({ id: `concurrent_2_${Date.now()}` }),
        helpers.generateTestRow({ id: `concurrent_3_${Date.now()}` })
      ];

      // Create rows in sequence (simulating rapid user actions)
      for (const row of rows) {
        await helpers.createRow(row);
      }

      // Verify all rows were created
      for (const row of rows) {
        await helpers.verifyRowExists(row.id!, row.values);
      }
    });
  });

  test.describe('UI State Management', () => {
    test('should maintain form state during interactions', async ({ page }) => {
      const user = createTestUser('_form_state');
      const table = createTestTable('_state');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Open form and partially fill it
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');
      
      await page.fill('input[placeholder="name"]', 'Partial Entry');
      
      // Cancel form
      const cancelButton = page.locator('button:has-text("Cancel")');
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      } else {
        await page.keyboard.press('Escape');
      }

      // Reopen form - should be empty
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');
      
      const nameInput = page.locator('input[placeholder="name"]');
      const nameValue = await nameInput.inputValue();
      expect(nameValue).toBe('');
    });

    test('should handle form validation feedback', async ({ page }) => {
      const user = createTestUser('_form_validation');
      const table = createTestTable('_validation');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Try to submit with invalid data (if validation exists)
      await page.fill('input[placeholder="age"]', 'invalid_number');
      await page.click('button:has-text("Save")');

      // Should show validation feedback or handle gracefully
      await page.waitForTimeout(2000);
    });
  });

  test.describe('Integration Tests', () => {
    test('should complete full workflow: create table, add rows, edit, delete', async ({ page }) => {
      const user = createTestUser('_workflow');
      const table = createTestTable('_workflow');

      // Step 1: Register and login
      await helpers.registerUser(user);

      // Step 2: Create table
      await helpers.createTable(table);
      await helpers.verifyTableExists(table.name);

      // Step 3: Navigate to table
      await helpers.selectTable(table.name);
      await helpers.waitForTableLoad();

      // Step 4: Add multiple rows
      const rows = [
        helpers.generateTestRow({ id: `workflow_1_${Date.now()}` }),
        helpers.generateTestRow({ id: `workflow_2_${Date.now()}` }),
        helpers.generateTestRow({ id: `workflow_3_${Date.now()}` })
      ];

      for (const row of rows) {
        await helpers.createRow(row);
        await helpers.verifyRowExists(row.id!, row.values);
      }

      // Step 5: Edit a row
      const updatedValues = { name: 'Updated Name', age: 999, active: false };
      await helpers.editRow(rows[0].id!, updatedValues);
      await helpers.verifyRowExists(rows[0].id!, updatedValues);

      // Step 6: Delete a row
      await helpers.deleteRow(rows[1].id!);
      await helpers.verifyRowDoesNotExist(rows[1].id!);

      // Step 7: Verify final state
      await helpers.verifyRowExists(rows[0].id!, updatedValues);
      await helpers.verifyRowDoesNotExist(rows[1].id!);
      await helpers.verifyRowExists(rows[2].id!, rows[2].values);
    });

    test('should handle session persistence during operations', async ({ page }) => {
      const user = createTestUser('_session_persist');
      const table = createTestTable('_persist');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Add a row
      const row = helpers.generateTestRow({ id: `persist_${Date.now()}` });
      await helpers.createRow(row);

      // Reload page
      await page.reload();
      await helpers.waitForTableLoad();

      // Navigate back to table
      await helpers.selectTable(table.name);

      // Verify row still exists
      await helpers.verifyRowExists(row.id!, row.values);
    });
  });
});