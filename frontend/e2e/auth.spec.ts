import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser } from './helpers/test-helpers';

test.describe('Authentication', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('User Registration', () => {
    test('should register a new user successfully', async ({ page }) => {
      const user = createTestUser('_reg');
      
      await page.goto('/');
      
      // Wait for auth form to load
      await page.waitForSelector('[data-testid="auth-form"]');
      
      // Switch to register tab
      await page.click('text=Register');
      
      // Fill registration form
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', user.email);
      await page.fill('input[placeholder="Password"]', user.password);
      
      // Submit registration
      await page.click('button:has-text("Register")');
      
      // Should redirect to main app
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible({ timeout: 15000 });
      
      // Should show user info or dashboard
      await expect(page.locator('text=Tables')).toBeVisible();
    });

    test('should show error for duplicate username', async ({ page }) => {
      const user = createTestUser('_dup');
      
      // Register user first time
      await helpers.registerUser(user);
      await helpers.logout();
      
      // Try to register same user again
      await page.goto('/');
      await page.click('text=Register');
      
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', 'different@example.com');
      await page.fill('input[placeholder="Password"]', user.password);
      
      await page.click('button:has-text("Register")');
      
      // Should show error message
      await expect(page.locator('text*=already exists')).toBeVisible({ timeout: 10000 });
    });

    test('should show error for invalid email format', async ({ page }) => {
      const user = createTestUser('_invalid');
      
      await page.goto('/');
      await page.click('text=Register');
      
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', 'invalid-email');
      await page.fill('input[placeholder="Password"]', user.password);
      
      await page.click('button:has-text("Register")');
      
      // Should show validation error
      await expect(page.locator('text*=email')).toBeVisible({ timeout: 5000 });
    });

    test('should show error for empty fields', async ({ page }) => {
      await page.goto('/');
      await page.click('text=Register');
      
      // Try to submit empty form
      await page.click('button:has-text("Register")');
      
      // Should show validation errors or prevent submission
      const isFormStillVisible = await page.locator('[data-testid="auth-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });
  });

  test.describe('User Login', () => {
    test('should login with valid credentials', async ({ page }) => {
      const user = createTestUser('_login');
      
      // Register user first
      await helpers.registerUser(user);
      await helpers.logout();
      
      // Now login
      await helpers.loginUser(user);
      
      // Should be in main app
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
      await expect(page.locator('text=Tables')).toBeVisible();
    });

    test('should show error for invalid credentials', async ({ page }) => {
      await page.goto('/');
      
      // Make sure we're on login tab
      const loginTab = page.locator('text=Login');
      if (await loginTab.isVisible()) {
        await loginTab.click();
      }
      
      await page.fill('input[placeholder="Username"]', 'nonexistentuser');
      await page.fill('input[placeholder="Password"]', 'wrongpassword');
      
      await page.click('button:has-text("Login")');
      
      // Should show error message
      await expect(page.locator('text*=Invalid')).toBeVisible({ timeout: 10000 });
    });

    test('should show error for empty login fields', async ({ page }) => {
      await page.goto('/');
      
      // Try to submit empty login form
      await page.click('button:has-text("Login")');
      
      // Should show validation errors or prevent submission
      const isFormStillVisible = await page.locator('[data-testid="auth-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });

    test('should switch between login and register tabs', async ({ page }) => {
      await page.goto('/');
      
      // Should start on login tab
      await expect(page.locator('button:has-text("Login")')).toBeVisible();
      
      // Switch to register
      await page.click('text=Register');
      await expect(page.locator('button:has-text("Register")')).toBeVisible();
      
      // Switch back to login
      await page.click('text=Login');
      await expect(page.locator('button:has-text("Login")')).toBeVisible();
    });
  });

  test.describe('Session Management', () => {
    test('should logout successfully', async ({ page }) => {
      const user = createTestUser('_logout');
      
      // Register and login user
      await helpers.registerUser(user);
      
      // Should be logged in
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
      
      // Logout
      await helpers.logout();
      
      // Should be back to auth form
      await expect(page.locator('[data-testid="auth-form"]')).toBeVisible();
    });

    test('should persist session across page reloads', async ({ page }) => {
      const user = createTestUser('_session');
      
      // Register user
      await helpers.registerUser(user);
      
      // Should be logged in
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
      
      // Reload page
      await page.reload();
      
      // Should still be logged in
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible({ timeout: 10000 });
    });

    test('should redirect to login when accessing protected routes without auth', async ({ page }) => {
      // Try to access app directly without authentication
      await page.goto('/');
      
      // Should show auth form
      await expect(page.locator('[data-testid="auth-form"]')).toBeVisible();
    });
  });

  test.describe('API Key Management', () => {
    test('should receive API key after registration', async ({ page }) => {
      const user = createTestUser('_apikey');
      
      await helpers.registerUser(user);
      
      // Should be logged in and have access to main app
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();
      
      // Check if we can access API-dependent features (like creating tables)
      await expect(page.locator('button:has-text("Create Table")')).toBeVisible();
    });

    test('should be able to make authenticated API calls', async ({ page }) => {
      const user = createTestUser('_api');
      
      await helpers.registerUser(user);
      
      // Try to load tables (requires API authentication)
      await page.waitForSelector('[data-testid="main-app"]');
      
      // Should not show API errors
      await helpers.expectNoErrors();
      
      // Should be able to see tables section
      await expect(page.locator('text=Tables')).toBeVisible();
    });
  });

  test.describe('Error Handling', () => {
    test('should handle network errors gracefully', async ({ page }) => {
      // Start with a valid registration attempt
      const user = createTestUser('_network');
      
      await page.goto('/');
      await page.click('text=Register');
      
      // Fill form
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', user.email);
      await page.fill('input[placeholder="Password"]', user.password);
      
      // Intercept and fail the network request
      await page.route('/api/auth/register', route => {
        route.abort();
      });
      
      await page.click('button:has-text("Register")');
      
      // Should show some kind of error or remain on the form
      const isFormStillVisible = await page.locator('[data-testid="auth-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });

    test('should handle server errors gracefully', async ({ page }) => {
      const user = createTestUser('_server');
      
      await page.goto('/');
      await page.click('text=Register');
      
      // Fill form
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', user.email);
      await page.fill('input[placeholder="Password"]', user.password);
      
      // Intercept and return server error
      await page.route('/api/auth/register', route => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal server error' })
        });
      });
      
      await page.click('button:has-text("Register")');
      
      // Should show error message
      await expect(page.locator('text*=error')).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Form Validation', () => {
    test('should validate password strength', async ({ page }) => {
      await page.goto('/');
      await page.click('text=Register');
      
      const user = createTestUser('_weak');
      await page.fill('input[placeholder="Username"]', user.username);
      await page.fill('input[placeholder="Email"]', user.email);
      await page.fill('input[placeholder="Password"]', '123'); // Weak password
      
      await page.click('button:has-text("Register")');
      
      // Should show validation error or prevent submission
      const isFormStillVisible = await page.locator('[data-testid="auth-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });

    test('should validate username format', async ({ page }) => {
      await page.goto('/');
      await page.click('text=Register');
      
      await page.fill('input[placeholder="Username"]', 'a'); // Too short
      await page.fill('input[placeholder="Email"]', 'test@example.com');
      await page.fill('input[placeholder="Password"]', 'validpassword123');
      
      await page.click('button:has-text("Register")');
      
      // Should show validation error or prevent submission
      const isFormStillVisible = await page.locator('[data-testid="auth-form"]').isVisible();
      expect(isFormStillVisible).toBe(true);
    });
  });
});