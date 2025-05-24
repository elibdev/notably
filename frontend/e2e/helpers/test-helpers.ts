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

  // Bulk operation helpers
  async createMultipleRows(rows: TestRow[]): Promise<void> {
    for (const row of rows) {
      await this.createRow(row);
      await this.page.waitForTimeout(200); // Brief pause between operations
    }
  }

  async updateMultipleRows(updates: Array<{ id: string; values: Record<string, unknown> }>): Promise<void> {
    for (const update of updates) {
      await this.editRow(update.id, update.values);
      await this.page.waitForTimeout(200);
    }
  }

  async deleteMultipleRows(rowIds: string[]): Promise<void> {
    for (const id of rowIds) {
      await this.deleteRow(id);
      await this.page.waitForTimeout(200);
    }
  }

  // Search and filtering helpers
  async searchTable(searchTerm: string): Promise<void> {
    const searchInput = this.page.locator('input[placeholder*="Search"], input[placeholder*="Filter"]');
    if (await searchInput.isVisible()) {
      await searchInput.fill(searchTerm);
      await this.page.waitForTimeout(1000); // Wait for filter to apply
    }
  }

  async clearSearch(): Promise<void> {
    const searchInput = this.page.locator('input[placeholder*="Search"], input[placeholder*="Filter"]');
    if (await searchInput.isVisible()) {
      await searchInput.clear();
      await this.page.waitForTimeout(500);
    }
  }

  async sortByColumn(columnName: string): Promise<void> {
    const columnHeader = this.page.locator(`th:has-text("${columnName}")`);
    if (await columnHeader.isVisible()) {
      await columnHeader.click();
      await this.page.waitForTimeout(1000);
    }
  }

  // Keyboard navigation helpers
  async navigateWithKeyboard(key: string, times: number = 1): Promise<void> {
    for (let i = 0; i < times; i++) {
      await this.page.keyboard.press(key);
      await this.page.waitForTimeout(100);
    }
  }

  async activateWithKeyboard(): Promise<void> {
    await this.page.keyboard.press('Enter');
  }

  async cancelWithEscape(): Promise<void> {
    await this.page.keyboard.press('Escape');
  }

  async toggleCheckboxWithKeyboard(): Promise<void> {
    await this.page.keyboard.press('Space');
  }

  // Performance helpers
  async measureOperationTime<T>(operation: () => Promise<T>): Promise<{ result: T; duration: number }> {
    const startTime = Date.now();
    const result = await operation();
    const duration = Date.now() - startTime;
    return { result, duration };
  }

  async waitForTableToStabilize(): Promise<void> {
    await this.waitForNetworkIdle();
    await this.page.waitForTimeout(1000); // Additional stability wait
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

  async expectErrorMessage(message: string): Promise<void> {
    const errorElement = this.page.locator(`text*=${message}, .error:has-text("${message}")`);
    await expect(errorElement).toBeVisible({ timeout: 10000 });
  }

  async simulateNetworkError(urlPattern: string): Promise<void> {
    await this.page.route(urlPattern, route => {
      route.abort();
    });
  }

  async removeNetworkErrorSimulation(urlPattern: string): Promise<void> {
    await this.page.unroute(urlPattern);
  }

  async capturePageState(): Promise<void> {
    // Take screenshot for debugging
    await this.page.screenshot({ path: `debug-${Date.now()}.png`, fullPage: true });
    
    // Log console messages
    console.log('Page URL:', this.page.url());
    console.log('Page title:', await this.page.title());
  }

  // Accessibility helpers
  async checkFocusVisible(): Promise<boolean> {
    const focusedElement = this.page.locator(':focus');
    return await focusedElement.isVisible();
  }

  async getFocusedElementText(): Promise<string | null> {
    const focusedElement = this.page.locator(':focus');
    if (await focusedElement.isVisible()) {
      return await focusedElement.textContent();
    }
    return null;
  }

  async checkAriaAttributes(selector: string): Promise<Record<string, string | null>> {
    const element = this.page.locator(selector);
    return await element.evaluate(el => {
      return {
        role: el.getAttribute('role'),
        'aria-label': el.getAttribute('aria-label'),
        'aria-describedby': el.getAttribute('aria-describedby'),
        'aria-live': el.getAttribute('aria-live'),
        'aria-expanded': el.getAttribute('aria-expanded'),
        'aria-hidden': el.getAttribute('aria-hidden')
      };
    });
  }

  async checkColorContrast(selector: string): Promise<{ color: string; backgroundColor: string; fontSize: string }> {
    const element = this.page.locator(selector);
    return await element.evaluate(el => {
      const styles = window.getComputedStyle(el);
      return {
        color: styles.color,
        backgroundColor: styles.backgroundColor,
        fontSize: styles.fontSize
      };
    });
  }

  // Modal and dialog helpers
  async waitForModal(modalSelector: string): Promise<void> {
    await this.page.waitForSelector(modalSelector, { timeout: 5000 });
  }

  async closeModalWithEscape(): Promise<void> {
    await this.page.keyboard.press('Escape');
    await this.page.waitForTimeout(500);
  }

  async verifyModalClosed(modalSelector: string): Promise<void> {
    await expect(this.page.locator(modalSelector)).not.toBeVisible();
  }

  // Form helpers enhancement
  async fillFormWithKeyboard(formData: Record<string, string>): Promise<void> {
    for (const [field, value] of Object.entries(formData)) {
      await this.page.keyboard.press('Tab');
      const currentField = this.page.locator(':focus');
      const placeholder = await currentField.getAttribute('placeholder');
      
      if (placeholder && placeholder.toLowerCase().includes(field.toLowerCase())) {
        await this.page.keyboard.type(value);
      }
    }
  }

  async submitFormWithKeyboard(): Promise<void> {
    // Navigate to submit button and activate
    for (let i = 0; i < 10; i++) {
      await this.page.keyboard.press('Tab');
      const focusedText = await this.getFocusedElementText();
      if (focusedText && (focusedText.includes('Save') || focusedText.includes('Create') || focusedText.includes('Submit'))) {
        await this.page.keyboard.press('Enter');
        break;
      }
    }
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

export function createCustomTestTable(suffix: string = '', columns: Array<{ name: string; dataType: string }>): TestTable {
  const timestamp = Date.now();
  return {
    name: `custom_table_${timestamp}${suffix}`,
    columns
  };
}

export function generateTestData(count: number, template: Record<string, unknown>): TestRow[] {
  const rows: TestRow[] = [];
  for (let i = 0; i < count; i++) {
    const values: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(template)) {
      if (typeof value === 'string') {
        values[key] = `${value} ${i + 1}`;
      } else if (typeof value === 'number') {
        values[key] = value + i;
      } else if (typeof value === 'boolean') {
        values[key] = i % 2 === 0;
      } else {
        values[key] = value;
      }
    }
    rows.push({ values });
  }
  return rows;
}

export async function setupTestEnvironment(page: Page): Promise<{ helpers: TestHelpers; user: TestUser; table: TestTable }> {
  const helpers = new TestHelpers(page);
  const user = createTestUser();
  const table = createTestTable();
  
  return { helpers, user, table };
}

export async function setupComplexTestEnvironment(page: Page, tableCount: number = 1): Promise<{
  helpers: TestHelpers;
  user: TestUser;
  tables: TestTable[];
}> {
  const helpers = new TestHelpers(page);
  const user = createTestUser();
  const tables: TestTable[] = [];
  
  for (let i = 0; i < tableCount; i++) {
    tables.push(createTestTable(`_${i}`));
  }
  
  return { helpers, user, tables };
}

// Utility functions for test data generation
export function createLargeDataset(size: number): TestRow[] {
  return generateTestData(size, {
    name: 'Test User',
    email: 'test@example.com',
    age: 25,
    active: true
  });
}

export function createDiverseDataset(): TestRow[] {
  return [
    { values: { name: 'Alice Johnson', email: 'alice@example.com', age: 25, active: true } },
    { values: { name: 'Bob Smith', email: 'bob@test.com', age: 30, active: false } },
    { values: { name: 'Charlie Brown', email: 'charlie@example.com', age: 35, active: true } },
    { values: { name: 'Diana Wilson', email: 'diana@sample.com', age: 28, active: false } },
    { values: { name: 'Eve Davis', email: 'eve@example.com', age: 32, active: true } }
  ];
}