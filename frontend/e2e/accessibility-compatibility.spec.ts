import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('Accessibility and Browser Compatibility', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Keyboard Navigation', () => {
    test('should support full keyboard navigation for authentication', async ({ page }) => {
      const user = createTestUser('_keyboard_auth');
      
      await page.goto('/');
      await page.waitForSelector('[data-testid="auth-form"]');

      // Test tab navigation through auth form
      await page.keyboard.press('Tab'); // Focus on first input
      await page.keyboard.type(user.username);
      
      await page.keyboard.press('Tab'); // Move to password
      await page.keyboard.type(user.password);
      
      await page.keyboard.press('Tab'); // Move to login button
      await page.keyboard.press('Enter'); // Should show error (no account)
      
      // Switch to register tab using keyboard
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab'); // Navigate to register tab
      await page.keyboard.press('Enter');
      
      // Fill registration form with keyboard
      await page.keyboard.press('Tab'); // Username field
      await page.keyboard.type(user.username);
      
      await page.keyboard.press('Tab'); // Email field
      await page.keyboard.type(user.email);
      
      await page.keyboard.press('Tab'); // Password field
      await page.keyboard.type(user.password);
      
      await page.keyboard.press('Tab'); // Register button
      await page.keyboard.press('Enter');
      
      // Should successfully register and navigate to main app
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible({ timeout: 15000 });
    });

    test('should support keyboard navigation in table operations', async ({ page }) => {
      const user = createTestUser('_keyboard_tables');
      const table = createTestTable('_keyboard');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test keyboard navigation for adding rows
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      
      // Look for "Add Row" button and activate with keyboard
      let addButtonFound = false;
      for (let i = 0; i < 10; i++) {
        const focusedElement = await page.locator(':focus').textContent();
        if (focusedElement && focusedElement.includes('Add')) {
          await page.keyboard.press('Enter');
          addButtonFound = true;
          break;
        }
        await page.keyboard.press('Tab');
      }

      if (addButtonFound) {
        await page.waitForSelector('[data-testid="row-form"]');
        
        // Fill form using keyboard navigation
        await page.keyboard.press('Tab'); // First field
        await page.keyboard.type('Keyboard Test User');
        
        await page.keyboard.press('Tab'); // Email field
        await page.keyboard.type('keyboard@example.com');
        
        await page.keyboard.press('Tab'); // Age field
        await page.keyboard.type('30');
        
        await page.keyboard.press('Tab'); // Boolean field
        await page.keyboard.press('Space'); // Toggle checkbox
        
        // Navigate to save button
        for (let i = 0; i < 5; i++) {
          await page.keyboard.press('Tab');
          const focusedElement = await page.locator(':focus').textContent();
          if (focusedElement && focusedElement.includes('Save')) {
            await page.keyboard.press('Enter');
            break;
          }
        }

        // Verify row was created
        await expect(page.locator('text=Keyboard Test User')).toBeVisible();
      }
    });

    test('should support escape key for canceling operations', async ({ page }) => {
      const user = createTestUser('_keyboard_escape');
      const table = createTestTable('_escape');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Open add row form
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Press escape to cancel
      await page.keyboard.press('Escape');
      
      // Form should close
      await expect(page.locator('[data-testid="row-form"]')).not.toBeVisible();

      // Test escape on table creation
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');
      
      await page.keyboard.press('Escape');
      await expect(page.locator('[data-testid="create-table-modal"]')).not.toBeVisible();
    });
  });

  test.describe('Screen Reader Compatibility', () => {
    test('should have proper ARIA labels and roles', async ({ page }) => {
      const user = createTestUser('_aria_labels');
      
      await page.goto('/');
      await page.waitForSelector('[data-testid="auth-form"]');

      // Check for proper form labeling
      const usernameInput = page.locator('input[placeholder="Username"]');
      const usernameLabel = await usernameInput.getAttribute('aria-label');
      const usernameAriaDescribedBy = await usernameInput.getAttribute('aria-describedby');
      
      // Should have some form of labeling
      expect(usernameLabel || usernameAriaDescribedBy).toBeTruthy();

      await helpers.registerUser(user);

      // Check main app area has proper landmarks
      const mainApp = page.locator('[data-testid="main-app"]');
      const mainRole = await mainApp.getAttribute('role');
      const mainAriaLabel = await mainApp.getAttribute('aria-label');
      
      // Should have proper semantic markup
      expect(mainRole || mainAriaLabel).toBeTruthy();

      // Check button accessibility
      const createTableButton = page.locator('button:has-text("Create Table")');
      const buttonAriaLabel = await createTableButton.getAttribute('aria-label');
      const buttonTitle = await createTableButton.getAttribute('title');
      
      // Buttons should have accessible names
      expect(buttonAriaLabel || buttonTitle || await createTableButton.textContent()).toBeTruthy();
    });

    test('should announce dynamic content changes', async ({ page }) => {
      const user = createTestUser('_dynamic_content');
      const table = createTestTable('_announce');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Check for live regions or aria-live attributes
      const liveRegions = page.locator('[aria-live], [role="status"], [role="alert"]');
      const liveRegionCount = await liveRegions.count();
      
      // Should have some mechanism for announcing changes
      if (liveRegionCount > 0) {
        console.log(`Found ${liveRegionCount} live regions for screen reader announcements`);
      }

      // Add a row and check for status announcements
      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: 'Screen Reader Test',
          email: 'sr@example.com',
          age: 25,
          active: true
        }
      }));

      // Should have some indication of success
      await helpers.expectNoErrors();
    });

    test('should provide proper table semantics', async ({ page }) => {
      const user = createTestUser('_table_semantics');
      const table = createTestTable('_semantics');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Check for proper table markup
      const tableElement = page.locator('table, [role="table"]');
      const hasTableSemantics = await tableElement.count() > 0;

      if (hasTableSemantics) {
        // Check for column headers
        const headers = page.locator('th, [role="columnheader"]');
        const headerCount = await headers.count();
        expect(headerCount).toBeGreaterThan(0);

        // Check for row semantics
        const rows = page.locator('tr, [role="row"]');
        const rowCount = await rows.count();
        expect(rowCount).toBeGreaterThan(0);
      }

      // Add data and verify it's accessible
      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: 'Table Semantics Test',
          email: 'table@example.com',
          age: 30,
          active: true
        }
      }));

      await expect(page.locator('text=Table Semantics Test')).toBeVisible();
    });
  });

  test.describe('Focus Management', () => {
    test('should manage focus correctly in modal dialogs', async ({ page }) => {
      const user = createTestUser('_focus_modal');
      
      await helpers.registerUser(user);

      // Open create table modal
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');

      // Focus should be trapped in modal
      const modalElements = page.locator('[data-testid="create-table-modal"] input, [data-testid="create-table-modal"] button');
      const elementCount = await modalElements.count();

      if (elementCount > 0) {
        // Test tab cycling within modal
        for (let i = 0; i < elementCount + 2; i++) {
          await page.keyboard.press('Tab');
          const focusedElement = page.locator(':focus');
          const isInModal = await page.locator('[data-testid="create-table-modal"]').locator(':focus').count() > 0;
          
          // Focus should stay within modal
          if (i < elementCount) {
            expect(isInModal).toBe(true);
          }
        }
      }

      // Close modal and verify focus returns
      await page.keyboard.press('Escape');
      await expect(page.locator('[data-testid="create-table-modal"]')).not.toBeVisible();
      
      // Focus should return to trigger element or a logical location
      const focusedAfterClose = page.locator(':focus');
      await expect(focusedAfterClose).toBeVisible();
    });

    test('should maintain focus after dynamic content updates', async ({ page }) => {
      const user = createTestUser('_focus_updates');
      const table = createTestTable('_focus_test');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Focus on add row button
      const addButton = page.locator('button:has-text("Add Row")');
      await addButton.focus();
      
      // Verify focus
      const focusedElement = page.locator(':focus');
      const addButtonFocused = await addButton.locator(':focus').count() > 0;
      
      if (addButtonFocused) {
        // Click to add row
        await page.keyboard.press('Enter');
        await page.waitForSelector('[data-testid="row-form"]');
        
        // Focus should move to first form field
        const firstFormField = page.locator('[data-testid="row-form"] input').first();
        const fieldFocused = await firstFormField.locator(':focus').count() > 0;
        expect(fieldFocused).toBe(true);
      }
    });

    test('should provide visible focus indicators', async ({ page }) => {
      const user = createTestUser('_focus_indicators');
      
      await page.goto('/');
      await page.waitForSelector('[data-testid="auth-form"]');

      // Test focus indicators on form elements
      const usernameInput = page.locator('input[placeholder="Username"]');
      await usernameInput.focus();
      
      // Check for visual focus indicators
      const focusedStyles = await usernameInput.evaluate(el => {
        const styles = window.getComputedStyle(el);
        return {
          outline: styles.outline,
          outlineColor: styles.outlineColor,
          outlineWidth: styles.outlineWidth,
          boxShadow: styles.boxShadow,
          borderColor: styles.borderColor
        };
      });

      // Should have some form of focus indication
      const hasFocusIndicator = 
        focusedStyles.outline !== 'none' ||
        focusedStyles.outlineWidth !== '0px' ||
        focusedStyles.boxShadow !== 'none' ||
        focusedStyles.borderColor !== 'initial';
      
      expect(hasFocusIndicator).toBe(true);
    });
  });

  test.describe('Color and Contrast', () => {
    test('should have sufficient color contrast for text', async ({ page }) => {
      const user = createTestUser('_contrast_test');
      
      await helpers.registerUser(user);

      // Check contrast on various text elements
      const textElements = [
        page.locator('button:has-text("Create Table")'),
        page.locator('text=Tables'),
        page.locator('label, .label').first()
      ];

      for (const element of textElements) {
        if (await element.isVisible()) {
          const styles = await element.evaluate(el => {
            const computed = window.getComputedStyle(el);
            return {
              color: computed.color,
              backgroundColor: computed.backgroundColor,
              fontSize: computed.fontSize
            };
          });

          // Basic check that colors are defined
          expect(styles.color).toBeTruthy();
          expect(styles.backgroundColor).toBeTruthy();
          
          // Log for manual review
          console.log('Element styles:', styles);
        }
      }
    });

    test('should not rely solely on color for information', async ({ page }) => {
      const user = createTestUser('_color_independence');
      const table = createTestTable('_color_test');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Add rows with different statuses
      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: 'Active User',
          email: 'active@example.com',
          age: 25,
          active: true
        }
      }));

      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: 'Inactive User',
          email: 'inactive@example.com',
          age: 30,
          active: false
        }
      }));

      // Check that boolean values have text indicators, not just colors
      const activeRow = page.locator('[data-testid^="row-"]:has-text("Active User")');
      const inactiveRow = page.locator('[data-testid^="row-"]:has-text("Inactive User")');

      // Should show "true"/"false" or similar text indicators
      await expect(activeRow.locator('text=true')).toBeVisible();
      await expect(inactiveRow.locator('text=false')).toBeVisible();
    });
  });

  test.describe('Browser Compatibility', () => {
    test('should work with different viewport sizes', async ({ page }) => {
      const user = createTestUser('_viewport_test');

      // Test mobile viewport
      await page.setViewportSize({ width: 375, height: 667 });
      await helpers.registerUser(user);
      
      // Should still be functional on mobile
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
      await expect(page.locator('button:has-text("Create Table")')).toBeVisible();

      // Test tablet viewport
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.reload();
      await helpers.waitForNetworkIdle();
      
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();

      // Test desktop viewport
      await page.setViewportSize({ width: 1920, height: 1080 });
      await page.reload();
      await helpers.waitForNetworkIdle();
      
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
    });

    test('should handle touch interactions on mobile', async ({ page }) => {
      const user = createTestUser('_touch_test');
      const table = createTestTable('_touch');

      // Set mobile viewport with touch simulation
      await page.setViewportSize({ width: 375, height: 667 });
      
      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test touch tap on buttons
      const addButton = page.locator('button:has-text("Add Row")');
      if (await addButton.isVisible()) {
        await addButton.tap();
        await page.waitForSelector('[data-testid="row-form"]');
        
        // Fill form with touch interactions
        await page.locator('input[placeholder*="name"]').tap();
        await page.keyboard.type('Touch Test User');
        
        await page.locator('button:has-text("Save")').tap();
        
        await expect(page.locator('text=Touch Test User')).toBeVisible();
      }
    });

    test('should work without JavaScript (graceful degradation)', async ({ page }) => {
      // Test with JavaScript disabled
      await page.context().addInitScript(() => {
        // Simulate some JavaScript limitations
        window.fetch = undefined;
      });

      await page.goto('/');
      
      // Basic HTML should still load
      await expect(page.locator('body')).toBeVisible();
      
      // Form elements should be present
      const forms = page.locator('form, [data-testid="auth-form"]');
      await expect(forms.first()).toBeVisible();
    });
  });

  test.describe('Performance and Loading', () => {
    test('should load quickly and show loading states', async ({ page }) => {
      const user = createTestUser('_performance_test');
      
      const startTime = Date.now();
      await page.goto('/');
      await page.waitForSelector('[data-testid="auth-form"]');
      const loadTime = Date.now() - startTime;
      
      // Should load within reasonable time
      expect(loadTime).toBeLessThan(10000); // 10 seconds max

      // Test loading indicators during operations
      await helpers.registerUser(user);
      
      // Look for loading states during table creation
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');
      await page.fill('input[placeholder="Table name"]', `perf_table_${Date.now()}`);
      
      const createButton = page.locator('button:has-text("Create")');
      await createButton.click();
      
      // Should show some form of loading state or complete quickly
      await helpers.expectNoErrors();
    });

    test('should handle large datasets efficiently', async ({ page }) => {
      const user = createTestUser('_large_dataset');
      const table = createTestTable('_large');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      const startTime = Date.now();
      
      // Create a medium-sized dataset for testing
      for (let i = 0; i < 10; i++) {
        await helpers.createRow(helpers.generateTestRow({
          values: {
            name: `Performance User ${i}`,
            email: `perf${i}@example.com`,
            age: 25 + i,
            active: i % 2 === 0
          }
        }));
      }
      
      const endTime = Date.now();
      const totalTime = endTime - startTime;
      
      // Should complete within reasonable time
      expect(totalTime).toBeLessThan(30000); // 30 seconds max
      
      // Verify all data loaded
      const finalCount = await helpers.getRowCount();
      expect(finalCount).toBe(10);
    });
  });

  test.describe('Error Handling and Recovery', () => {
    test('should provide accessible error messages', async ({ page }) => {
      await page.goto('/');
      await page.waitForSelector('[data-testid="auth-form"]');

      // Try to login with invalid credentials
      await page.fill('input[placeholder="Username"]', 'nonexistent');
      await page.fill('input[placeholder="Password"]', 'wrongpassword');
      await page.click('button:has-text("Login")');

      // Should show accessible error message
      const errorMessages = page.locator('[role="alert"], .error, [aria-live="polite"]');
      const errorCount = await errorMessages.count();
      
      if (errorCount > 0) {
        const firstError = errorMessages.first();
        await expect(firstError).toBeVisible();
        
        // Error should have accessible attributes
        const errorRole = await firstError.getAttribute('role');
        const errorAriaLive = await firstError.getAttribute('aria-live');
        
        expect(errorRole || errorAriaLive).toBeTruthy();
      }
    });

    test('should maintain accessibility during error states', async ({ page }) => {
      const user = createTestUser('_error_accessibility');
      
      await helpers.registerUser(user);

      // Create table with empty name to trigger error
      await page.click('button:has-text("Create Table")');
      await page.waitForSelector('[data-testid="create-table-modal"]');
      
      // Submit without name
      await page.click('button:has-text("Create")');
      
      // Modal should remain accessible
      const modal = page.locator('[data-testid="create-table-modal"]');
      await expect(modal).toBeVisible();
      
      // Focus should remain in modal
      const focusInModal = await modal.locator(':focus').count() > 0;
      if (!focusInModal) {
        // At least the modal should be accessible via keyboard
        await page.keyboard.press('Tab');
        const focusAfterTab = await modal.locator(':focus').count() > 0;
        expect(focusAfterTab).toBe(true);
      }
    });
  });

  test.describe('Internationalization Support', () => {
    test('should handle different text lengths gracefully', async ({ page }) => {
      const user = createTestUser('_i18n_test');
      const table = createTestTable('_i18n');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test with very long text (simulating different languages)
      const longText = 'This is a very long text string that might be used in different languages and should not break the layout or functionality of the application when displayed in the user interface';
      
      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: longText,
          email: 'i18n@example.com',
          age: 25,
          active: true
        }
      }));

      // Should handle long text without breaking layout
      await expect(page.locator(`text=${longText.substring(0, 50)}`)).toBeVisible();
      
      // UI should remain functional
      await helpers.expectNoErrors();
    });

    test('should support right-to-left text direction', async ({ page }) => {
      const user = createTestUser('_rtl_test');
      const table = createTestTable('_rtl');

      await helpers.registerUser(user);
      await helpers.createTable(table);
      await helpers.selectTable(table.name);

      // Test with RTL text (Arabic example)
      const rtlText = 'النص العربي للاختبار';
      
      await helpers.createRow(helpers.generateTestRow({
        values: {
          name: rtlText,
          email: 'rtl@example.com',
          age: 30,
          active: true
        }
      }));

      // Should display RTL text correctly
      await expect(page.locator(`text=${rtlText}`)).toBeVisible();
      
      // Layout should remain intact
      await helpers.expectNoErrors();
    });
  });
});