import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('Regression Tests - Row Creation Fixes', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Row Creation Bug Fixes', () => {
    test('should create row without ID when ID field is empty', async ({ page }) => {
      const user = createTestUser('_reg_empty_id');
      const table = createTestTable('_empty_id');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();

      // Open row creation form
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Leave ID field empty and fill other fields
      const idInput = page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.clear(); // Ensure it's empty
      }

      await page.fill('input[placeholder="name"]', 'Test User');
      await page.fill('input[placeholder="email"]', 'test@example.com');
      await page.fill('input[placeholder="age"]', '25');

      // Submit the form
      await page.click('button:has-text("Save")');

      // Wait for form to close and row to appear
      await page.waitForTimeout(2000);

      // Verify row was created successfully
      const newCount = await helpers.getRowCount();
      expect(newCount).toBe(initialCount + 1);

      // Verify the data appears in the table
      await expect(page.locator('text=Test User')).toBeVisible();
      await expect(page.locator('text=test@example.com')).toBeVisible();
      await expect(page.locator('text=25')).toBeVisible();
    });

    test('should create row when ID field contains only whitespace', async ({ page }) => {
      const user = createTestUser('_reg_whitespace_id');
      const table = createTestTable('_whitespace');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Fill ID with whitespace
      const idInput = page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.fill('   '); // Just spaces
      }

      await page.fill('input[placeholder="name"]', 'Whitespace Test');
      await page.fill('input[placeholder="email"]', 'whitespace@example.com');
      await page.fill('input[placeholder="age"]', '30');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Should create row (backend should generate ID)
      const newCount = await helpers.getRowCount();
      expect(newCount).toBe(initialCount + 1);

      await expect(page.locator('text=Whitespace Test')).toBeVisible();
    });

    test('should handle undefined/null ID values correctly', async ({ page }) => {
      const user = createTestUser('_reg_null_id');
      const table = createTestTable('_null_id');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test multiple scenarios that could result in undefined/null IDs
      const testCases = [
        { name: 'No ID Input', skipId: true },
        { name: 'Empty ID', id: '' },
        { name: 'Null-like ID', id: 'null' },
        { name: 'Undefined-like ID', id: 'undefined' }
      ];

      for (const testCase of testCases) {
        const initialCount = await helpers.getRowCount();

        await page.click('button:has-text("Add Row")');
        await page.waitForSelector('[data-testid="row-form"]');

        if (!testCase.skipId) {
          const idInput = page.locator('input[placeholder="Row ID (optional)"]');
          if (await idInput.isVisible()) {
            await idInput.clear();
            if (testCase.id) {
              await idInput.fill(testCase.id);
            }
          }
        }

        await page.fill('input[placeholder="name"]', testCase.name);
        await page.fill('input[placeholder="email"]', `${testCase.name.toLowerCase().replace(' ', '')}@example.com`);
        await page.fill('input[placeholder="age"]', '25');

        await page.click('button:has-text("Save")');
        await page.waitForTimeout(2000);

        // Should create row successfully
        const newCount = await helpers.getRowCount();
        expect(newCount).toBe(initialCount + 1);

        await expect(page.locator(`text=${testCase.name}`)).toBeVisible();
      }
    });

    test('should properly reset form state between row creations', async ({ page }) => {
      const user = createTestUser('_reg_form_reset');
      const table = createTestTable('_form_reset');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Create first row with custom ID
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      const idInput = page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.fill('custom-id-1');
      }

      await page.fill('input[placeholder="name"]', 'First User');
      await page.fill('input[placeholder="email"]', 'first@example.com');
      await page.fill('input[placeholder="age"]', '25');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Create second row without specifying ID
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Verify form is clean
      if (await idInput.isVisible()) {
        const idValue = await idInput.inputValue();
        expect(idValue).toBe('');
      }

      const nameValue = await page.locator('input[placeholder="name"]').inputValue();
      expect(nameValue).toBe('');

      // Fill only required fields
      await page.fill('input[placeholder="name"]', 'Second User');
      await page.fill('input[placeholder="email"]', 'second@example.com');
      await page.fill('input[placeholder="age"]', '30');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Both rows should exist
      await expect(page.locator('text=First User')).toBeVisible();
      await expect(page.locator('text=Second User')).toBeVisible();
    });

    test('should handle rapid successive row creations', async ({ page }) => {
      const user = createTestUser('_reg_rapid');
      const table = createTestTable('_rapid');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();

      // Create multiple rows quickly
      for (let i = 0; i < 3; i++) {
        await page.click('button:has-text("Add Row")');
        await page.waitForSelector('[data-testid="row-form"]');

        await page.fill('input[placeholder="name"]', `Rapid User ${i}`);
        await page.fill('input[placeholder="email"]', `rapid${i}@example.com`);
        await page.fill('input[placeholder="age"]', String(20 + i));

        await page.click('button:has-text("Save")');
        await page.waitForTimeout(500); // Short wait between creations
      }

      // All rows should be created
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(initialCount + 3);

      // Verify all rows exist
      for (let i = 0; i < 3; i++) {
        await expect(page.locator(`text=Rapid User ${i}`)).toBeVisible();
      }
    });

    test('should handle form cancellation without side effects', async ({ page }) => {
      const user = createTestUser('_reg_cancel');
      const table = createTestTable('_cancel');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const initialCount = await helpers.getRowCount();

      // Open form and fill it
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      await page.fill('input[placeholder="name"]', 'Cancelled User');
      await page.fill('input[placeholder="email"]', 'cancelled@example.com');
      await page.fill('input[placeholder="age"]', '25');

      // Cancel the form
      const cancelButton = page.locator('button:has-text("Cancel")');
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      } else {
        await page.keyboard.press('Escape');
      }

      await page.waitForTimeout(1000);

      // No new row should be created
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(initialCount);

      // Data should not appear in table
      await expect(page.locator('text=Cancelled User')).not.toBeVisible();
    });

    test('should handle network interruption during row creation', async ({ page }) => {
      const user = createTestUser('_reg_network');
      const table = createTestTable('_network_test');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Set up network interception to fail the first request
      let requestCount = 0;
      await page.route('/api/tables/*/rows', route => {
        requestCount++;
        if (requestCount === 1) {
          // Fail the first request
          route.abort();
        } else {
          // Allow subsequent requests
          route.continue();
        }
      });

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      await page.fill('input[placeholder="name"]', 'Network Test');
      await page.fill('input[placeholder="email"]', 'network@example.com');
      await page.fill('input[placeholder="age"]', '25');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(3000);

      // Form should still be visible due to network error
      const isFormVisible = await page.locator('[data-testid="row-form"]').isVisible();
      expect(isFormVisible).toBe(true);

      // Try again - this should succeed
      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Row should be created successfully
      await expect(page.locator('text=Network Test')).toBeVisible();
    });

    test('should preserve form data during temporary UI state changes', async ({ page }) => {
      const user = createTestUser('_reg_preserve');
      const table = createTestTable('_preserve');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Fill form partially
      await page.fill('input[placeholder="name"]', 'Preserved User');
      await page.fill('input[placeholder="email"]', 'preserved@example.com');

      // Simulate some UI interaction that shouldn't affect form state
      // (like clicking somewhere else, hovering, etc.)
      await page.click('body'); // Click outside the form
      await page.waitForTimeout(500);

      // Data should still be there
      const nameValue = await page.locator('input[placeholder="name"]').inputValue();
      const emailValue = await page.locator('input[placeholder="email"]').inputValue();

      expect(nameValue).toBe('Preserved User');
      expect(emailValue).toBe('preserved@example.com');

      // Complete and submit
      await page.fill('input[placeholder="age"]', '25');
      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      await expect(page.locator('text=Preserved User')).toBeVisible();
    });

    test('should handle edge case column values during row creation', async ({ page }) => {
      const user = createTestUser('_reg_edge');
      const table = {
        name: `edge_case_table_${Date.now()}`,
        columns: [
          { name: 'text_field', dataType: 'string' },
          { name: 'number_field', dataType: 'number' },
          { name: 'bool_field', dataType: 'boolean' }
        ]
      };

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test edge case values
      const edgeCases = [
        {
          text_field: '',
          number_field: '0',
          bool_field: 'false',
          name: 'Empty/Zero/False'
        },
        {
          text_field: 'Very long text that might cause issues with the UI layout and form validation and data processing',
          number_field: '999999999',
          bool_field: 'true',
          name: 'Long/Large/True'
        },
        {
          text_field: 'Special chars: !@#$%^&*()_+-=[]{}|;:,.<>?',
          number_field: '-42',
          bool_field: 'false',
          name: 'Special/Negative/False'
        }
      ];

      for (const edgeCase of edgeCases) {
        await page.click('button:has-text("Add Row")');
        await page.waitForSelector('[data-testid="row-form"]');

        await page.fill('input[placeholder="text_field"]', edgeCase.text_field);
        await page.fill('input[placeholder="number_field"]', edgeCase.number_field);
        await page.fill('input[placeholder="bool_field"]', edgeCase.bool_field);

        await page.click('button:has-text("Save")');
        await page.waitForTimeout(2000);

        // Verify row was created (checking for unique identifiers)
        if (edgeCase.text_field) {
          await expect(page.locator(`text=${edgeCase.text_field}`)).toBeVisible();
        }
        await expect(page.locator(`text=${edgeCase.number_field}`)).toBeVisible();
      }
    });
  });

  test.describe('API Payload Validation', () => {
    test('should send correct payload structure for row creation', async ({ page }) => {
      const user = createTestUser('_reg_payload');
      const table = createTestTable('_payload');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Intercept the API call to verify payload structure
      let capturedPayload: any = null;
      await page.route('/api/tables/*/rows', async route => {
        const request = route.request();
        const postData = request.postData();
        if (postData) {
          capturedPayload = JSON.parse(postData);
        }
        route.continue();
      });

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Create row without ID
      await page.fill('input[placeholder="name"]', 'Payload Test');
      await page.fill('input[placeholder="email"]', 'payload@example.com');
      await page.fill('input[placeholder="age"]', '25');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Verify the payload structure
      expect(capturedPayload).toBeTruthy();
      expect(capturedPayload).toHaveProperty('values');
      expect(capturedPayload.values).toHaveProperty('name', 'Payload Test');
      expect(capturedPayload.values).toHaveProperty('email', 'payload@example.com');
      expect(capturedPayload.values).toHaveProperty('age', 25); // Should be converted to number

      // Should not have an 'id' property or it should be undefined/empty
      if ('id' in capturedPayload) {
        expect(capturedPayload.id).toBeFalsy();
      }
    });

    test('should send correct payload when ID is provided', async ({ page }) => {
      const user = createTestUser('_reg_payload_id');
      const table = createTestTable('_payload_id');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      let capturedPayload: any = null;
      await page.route('/api/tables/*/rows', async route => {
        const request = route.request();
        const postData = request.postData();
        if (postData) {
          capturedPayload = JSON.parse(postData);
        }
        route.continue();
      });

      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      const customId = `custom_${Date.now()}`;
      const idInput = page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.fill(customId);
      }

      await page.fill('input[placeholder="name"]', 'Payload With ID');
      await page.fill('input[placeholder="email"]', 'payloadid@example.com');
      await page.fill('input[placeholder="age"]', '30');

      await page.click('button:has-text("Save")');
      await page.waitForTimeout(2000);

      // Verify the payload includes the ID
      expect(capturedPayload).toBeTruthy();
      expect(capturedPayload).toHaveProperty('id', customId);
      expect(capturedPayload).toHaveProperty('values');
      expect(capturedPayload.values).toHaveProperty('name', 'Payload With ID');
    });
  });
});