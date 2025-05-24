import { Page, expect } from '@playwright/test';

export interface TestUser {
  username: string;
  email: string;
  password: string;
  apiKey?: string;
}

export interface TestTable {
  name: string;
  columns?: Array<{ name: string; dataType: string }>;
}

export interface TestRow {
  id?: string;
  values: Record<string, unknown>;
}

export class TestHelpers {
  constructor(private page: Page) {}

  // Authentication helpers
  async registerUser(user: TestUser): Promise<void> {
    await this.page.goto('/');
    
    // Wait for auth form to load
    await this.page.waitForSelector('[data-testid="auth-form"]', { timeout: 10000 });
    
    // Switch to register tab if not already there
    const registerTab = this.page.locator('text=Register');
    if (await registerTab.isVisible()) {
      await registerTab.click();
    }

    // Fill registration form
    await this.page.fill('input[placeholder="Username"]', user.username);
    await this.page.fill('input[placeholder="Email"]', user.email);
    await this.page.fill('input[placeholder="Password"]', user.password);
    
    // Submit registration
    await this.page.click('button:has-text("Register")');
    
    // Wait for successful registration (should redirect to main app)
    await this.page.waitForSelector('[data-testid="main-app"]', { timeout: 15000 });
  }

  async loginUser(user: TestUser): Promise<void> {
    await this.page.goto('/');
    
    // Wait for auth form to load
    await this.page.waitForSelector('[data-testid="auth-form"]', { timeout: 10000 });
    
    // Make sure we're on login tab
    const loginTab = this.page.locator('text=Login');
    if (await loginTab.isVisible()) {
      await loginTab.click();
    }

    // Fill login form
    await this.page.fill('input[placeholder="Username"]', user.username);
    await this.page.fill('input[placeholder="Password"]', user.password);
    
    // Submit login
    await this.page.click('button:has-text("Login")');
    
    // Wait for successful login
    await this.page.waitForSelector('[data-testid="main-app"]', { timeout: 15000 });
  }

  async logout(): Promise<void> {
    const logoutButton = this.page.locator('button:has-text("Logout")');
    if (await logoutButton.isVisible()) {
      await logoutButton.click();
    }
    
    // Wait for auth form to appear
    await this.page.waitForSelector('[data-testid="auth-form"]', { timeout: 10000 });
  }

  // Table management helpers
  async createTable(table: TestTable): Promise<void> {
    // Click create table button
    await this.page.click('button:has-text("Create Table")');
    
    // Wait for modal to open
    await this.page.waitForSelector('[data-testid="create-table-modal"]', { timeout: 5000 });
    
    // Fill table name
    await this.page.fill('input[placeholder="Table name"]', table.name);
    
    // Add columns if specified
    if (table.columns && table.columns.length > 0) {
      for (const column of table.columns) {
        await this.page.click('button:has-text("Add Column")');
        
        // Fill column details in the last row
        const columnRows = this.page.locator('[data-testid="column-row"]');
        const lastRow = columnRows.last();
        
        await lastRow.locator('input[placeholder="Column name"]').fill(column.name);
        await lastRow.locator('select').selectOption(column.dataType);
      }
    }
    
    // Submit table creation
    await this.page.click('button:has-text("Create")');
    
    // Wait for modal to close and table to appear in list
    await this.page.waitForSelector(`text=${table.name}`, { timeout: 10000 });
  }

  async selectTable(tableName: string): Promise<void> {
    await this.page.click(`text=${tableName}`);
    
    // Wait for table view to load
    await this.page.waitForSelector('[data-testid="table-view"]', { timeout: 10000 });
  }

  // Row management helpers
  async createRow(row: TestRow): Promise<void> {
    // Click add row button
    await this.page.click('button:has-text("Add Row")');
    
    // Wait for row form to appear
    await this.page.waitForSelector('[data-testid="row-form"]', { timeout: 5000 });
    
    // Fill ID if provided
    if (row.id) {
      const idInput = this.page.locator('input[placeholder="Row ID (optional)"]');
      if (await idInput.isVisible()) {
        await idInput.fill(row.id);
      }
    }
    
    // Fill column values
    for (const [columnName, value] of Object.entries(row.values)) {
      const input = this.page.locator(`input[placeholder="${columnName}"]`);
      await input.fill(String(value));
    }
    
    // Submit row creation
    await this.page.click('button:has-text("Save")');
    
    // Wait for row to appear in table
    await this.page.waitForTimeout(1000); // Brief wait for UI update
  }

  async editRow(rowId: string, newValues: Record<string, unknown>): Promise<void> {
    // Find and click edit button for the specific row
    const editButton = this.page.locator(`[data-testid="row-${rowId}"] button:has-text("Edit")`);
    await editButton.click();
    
    // Wait for edit form
    await this.page.waitForSelector('[data-testid="row-form"]', { timeout: 5000 });
    
    // Update values
    for (const [columnName, value] of Object.entries(newValues)) {
      const input = this.page.locator(`input[placeholder="${columnName}"]`);
      await input.clear();
      await input.fill(String(value));
    }
    
    // Save changes
    await this.page.click('button:has-text("Save")');
    
    // Wait for form to close
    await this.page.waitForTimeout(1000);
  }

  async deleteRow(rowId: string): Promise<void> {
    // Find and click delete button for the specific row
    const deleteButton = this.page.locator(`[data-testid="row-${rowId}"] button:has-text("Delete")`);
    await deleteButton.click();
    
    // Confirm deletion if confirmation dialog appears
    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible({ timeout: 2000 })) {
      await confirmButton.click();
    }
    
    // Wait for row to disappear
    await this.page.waitForTimeout(1000);
  }

  // Verification helpers
  async verifyTableExists(tableName: string): Promise<void> {
    await expect(this.page.locator(`text=${tableName}`)).toBeVisible();
  }

  async verifyRowExists(rowId: string, expectedValues?: Record<string, unknown>): Promise<void> {
    const rowElement = this.page.locator(`[data-testid="row-${rowId}"]`);
    await expect(rowElement).toBeVisible();
    
    if (expectedValues) {
      for (const [, value] of Object.entries(expectedValues)) {
        await expect(rowElement.locator(`text=${String(value)}`)).toBeVisible();
      }
    }
  }

  async verifyRowDoesNotExist(rowId: string): Promise<void> {
    const rowElement = this.page.locator(`[data-testid="row-${rowId}"]`);
    await expect(rowElement).not.toBeVisible();
  }

  async verifyNotificationMessage(message: string): Promise<void> {
    await expect(this.page.locator(`text=${message}`)).toBeVisible({ timeout: 5000 });
  }

  async verifyErrorMessage(message: string): Promise<void> {
    await expect(this.page.locator('.error').locator(`text=${message}`)).toBeVisible({ timeout: 5000 });
  }

  // Wait helpers
  async waitForNetworkIdle(): Promise<void> {
    await this.page.waitForLoadState('networkidle');
  }

  async waitForTableLoad(): Promise<void> {
    await this.page.waitForSelector('[data-testid="table-view"]', { timeout: 10000 });
    await this.waitForNetworkIdle();
  }

  // Data helpers
  generateTestUser(suffix: string = ''): TestUser {
    const timestamp = Date.now();
    return {
      username: `testuser${timestamp}${suffix}`,
      email: `test${timestamp}${suffix}@example.com`,
      password: 'testpassword123'
    };
  }

  generateTestTable(suffix: string = ''): TestTable {
    const timestamp = Date.now();
    return {
      name: `test_table_${timestamp}${suffix}`,
      columns: [
        { name: 'name', dataType: 'string' },
        { name: 'age', dataType: 'number' },
        { name: 'active', dataType: 'boolean' }
      ]
    };
  }

  generateTestRow(overrides: Partial<TestRow> = {}): TestRow {
    const timestamp = Date.now();
    return {
      values: {
        name: `Test User ${timestamp}`,
        age: Math.floor(Math.random() * 80) + 18,
        active: Math.random() > 0.5
      },
      ...overrides
    };
  }

  // Cleanup helpers
  async cleanupTestData(): Promise<void> {
    // This would ideally make API calls to clean up test data
    // For now, we'll rely on unique names and manual cleanup
    console.log('Test cleanup - consider implementing API cleanup calls');
  }

  // Navigation helpers
  async navigateToTable(tableName: string): Promise<void> {
    await this.selectTable(tableName);
    await this.waitForTableLoad();
  }

  async navigateToHome(): Promise<void> {
    await this.page.goto('/');
    await this.page.waitForSelector('[data-testid="main-app"]', { timeout: 10000 });
  }

  // Form helpers
  async fillForm(formData: Record<string, string>): Promise<void> {
    for (const [field, value] of Object.entries(formData)) {
      await this.page.fill(`input[placeholder*="${field}"], input[name="${field}"]`, value);
    }
  }

  async submitForm(): Promise<void> {
    await this.page.click('button[type="submit"], button:has-text("Save"), button:has-text("Create")');
  }

  // Advanced verification helpers
  async verifyTableStructure(tableName: string, expectedColumns: string[]): Promise<void> {
    await this.selectTable(tableName);
    
    for (const columnName of expectedColumns) {
      await expect(this.page.locator(`th:has-text("${columnName}")`)).toBeVisible();
    }
  }

  async verifyRowCount(expectedCount: number): Promise<void> {
    const rows = this.page.locator('[data-testid^="row-"]');
    await expect(rows).toHaveCount(expectedCount);
  }

  async getRowCount(): Promise<number> {
    const rows = this.page.locator('[data-testid^="row-"]');
    return await rows.count();
  }

  // Error handling helpers
  async expectNoErrors(): Promise<void> {
    const errorElements = this.page.locator('.error, [data-testid="error"]');
    await expect(errorElements).toHaveCount(0);
  }

  async capturePageState(): Promise<void> {
    // Take screenshot for debugging
    await this.page.screenshot({ path: `debug-${Date.now()}.png`, fullPage: true });
    
    // Log console messages
    console.log('Page URL:', this.page.url());
    console.log('Page title:', await this.page.title());
  }
}

// Global test utilities
export function createTestUser(suffix: string = ''): TestUser {
  const timestamp = Date.now();
  return {
    username: `testuser${timestamp}${suffix}`,
    email: `test${timestamp}${suffix}@example.com`,
    password: 'testpassword123'
  };
}

export function createTestTable(suffix: string = ''): TestTable {
  const timestamp = Date.now();
  return {
    name: `test_table_${timestamp}${suffix}`,
    columns: [
      { name: 'name', dataType: 'string' },
      { name: 'email', dataType: 'string' },
      { name: 'age', dataType: 'number' },
      { name: 'active', dataType: 'boolean' }
    ]
  };
}

export async function setupTestEnvironment(page: Page): Promise<{ helpers: TestHelpers; user: TestUser; table: TestTable }> {
  const helpers = new TestHelpers(page);
  const user = createTestUser();
  const table = createTestTable();
  
  return { helpers, user, table };
}