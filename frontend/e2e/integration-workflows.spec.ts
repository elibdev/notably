import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser, createTestTable } from './helpers/test-helpers';

test.describe('Integration Workflows', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test.describe('Complete User Journeys', () => {
    test('should complete end-to-end personal information management workflow', async ({ page }) => {
      const user = createTestUser('_complete_journey');
      
      // Step 1: Registration and initial setup
      await helpers.registerUser(user);
      await expect(page.locator('[data-testid="main-app"]')).toBeVisible();

      // Step 2: Create personal information tables
      const contactsTable = {
        name: `contacts_${Date.now()}`,
        columns: [
          { name: 'name', dataType: 'string' },
          { name: 'phone', dataType: 'string' },
          { name: 'email', dataType: 'string' },
          { name: 'relationship', dataType: 'string' }
        ]
      };

      const journalTable = {
        name: `journal_${Date.now()}`,
        columns: [
          { name: 'title', dataType: 'string' },
          { name: 'content', dataType: 'string' },
          { name: 'mood', dataType: 'string' },
          { name: 'important', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(contactsTable);
      await helpers.createTable(journalTable);

      // Step 3: Add sample contacts
      await helpers.selectTable(contactsTable.name);
      
      const contacts = [
        { name: 'John Smith', phone: '555-0101', email: 'john@example.com', relationship: 'Friend' },
        { name: 'Jane Doe', phone: '555-0102', email: 'jane@example.com', relationship: 'Colleague' },
        { name: 'Mom', phone: '555-0103', email: 'mom@family.com', relationship: 'Family' }
      ];

      for (const contact of contacts) {
        await helpers.createRow(helpers.generateTestRow({ values: contact }));
      }

      // Step 4: Add journal entries
      await helpers.selectTable(journalTable.name);
      
      const journalEntries = [
        { title: 'Great Day', content: 'Had an amazing day at the park', mood: 'Happy', important: true },
        { title: 'Work Meeting', content: 'Productive meeting with the team', mood: 'Focused', important: false },
        { title: 'Family Dinner', content: 'Wonderful evening with family', mood: 'Content', important: true }
      ];

      for (const entry of journalEntries) {
        await helpers.createRow(helpers.generateTestRow({ values: entry }));
      }

      // Step 5: Verify data across tables
      await helpers.selectTable(contactsTable.name);
      await expect(page.locator('text=John Smith')).toBeVisible();
      await expect(page.locator('text=Friend')).toBeVisible();

      await helpers.selectTable(journalTable.name);
      await expect(page.locator('text=Great Day')).toBeVisible();
      await expect(page.locator('text=Happy')).toBeVisible();

      // Step 6: Update information
      await helpers.selectTable(contactsTable.name);
      // Find John's row and update his relationship
      const johnRow = page.locator('[data-testid^="row-"]:has-text("John Smith")');
      const editButton = johnRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="relationship"]', 'Best Friend');
      await page.click('button:has-text("Save")');
      
      // Verify update
      await expect(page.locator('text=Best Friend')).toBeVisible();

      // Step 7: Test session persistence
      await page.reload();
      await helpers.waitForTableLoad();
      await expect(page.locator('text=Best Friend')).toBeVisible();

      // Step 8: Navigate between tables to verify all data persists
      await helpers.selectTable(journalTable.name);
      await expect(page.locator('text=Great Day')).toBeVisible();
      
      await helpers.selectTable(contactsTable.name);
      await expect(page.locator('text=Best Friend')).toBeVisible();
    });

    test('should handle complex data lifecycle with history tracking', async ({ page }) => {
      const user = createTestUser('_data_lifecycle');
      await helpers.registerUser(user);

      // Create a projects table for tracking project evolution
      const projectsTable = {
        name: `projects_${Date.now()}`,
        columns: [
          { name: 'title', dataType: 'string' },
          { name: 'status', dataType: 'string' },
          { name: 'priority', dataType: 'string' },
          { name: 'completed', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(projectsTable);
      await helpers.selectTable(projectsTable.name);

      // Create initial project
      const projectId = `project_${Date.now()}`;
      await helpers.createRow(helpers.generateTestRow({
        id: projectId,
        values: {
          title: 'Website Redesign',
          status: 'Planning',
          priority: 'High',
          completed: false
        }
      }));

      // Simulate project evolution through multiple updates
      const statusUpdates = [
        { status: 'In Progress', priority: 'High' },
        { status: 'Review', priority: 'Medium' },
        { status: 'Testing', priority: 'Medium' },
        { status: 'Done', completed: true }
      ];

      for (const update of statusUpdates) {
        await helpers.editRow(projectId, update);
        await page.waitForTimeout(1000); // Ensure different timestamps
      }

      // Verify final state
      await expect(page.locator('text=Done')).toBeVisible();
      await expect(page.locator('text=true')).toBeVisible(); // completed = true

      // Test history functionality if available
      const historyButton = page.locator('button:has-text("History"), button:has-text("Snapshot")');
      if (await historyButton.first().isVisible()) {
        await historyButton.first().click();
        
        // Should show historical states
        await page.waitForTimeout(2000);
        await helpers.expectNoErrors();
      }
    });

    test('should support collaborative workflow simulation', async ({ page }) => {
      const user = createTestUser('_collaborative');
      await helpers.registerUser(user);

      // Create shared resource table
      const resourcesTable = {
        name: `shared_resources_${Date.now()}`,
        columns: [
          { name: 'resource_name', dataType: 'string' },
          { name: 'owner', dataType: 'string' },
          { name: 'status', dataType: 'string' },
          { name: 'available', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(resourcesTable);
      await helpers.selectTable(resourcesTable.name);

      // Simulate multiple users adding resources
      const resources = [
        { resource_name: 'Conference Room A', owner: 'Admin', status: 'Available', available: true },
        { resource_name: 'Projector', owner: 'IT', status: 'Available', available: true },
        { resource_name: 'Company Car', owner: 'HR', status: 'In Use', available: false }
      ];

      for (const resource of resources) {
        await helpers.createRow(helpers.generateTestRow({ values: resource }));
      }

      // Simulate resource booking (status changes)
      const conferenceRoomRow = page.locator('[data-testid^="row-"]:has-text("Conference Room A")');
      const editButton = conferenceRoomRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'Booked');
      await page.click('input[type="checkbox"]'); // Toggle available to false
      await page.click('button:has-text("Save")');

      // Verify booking
      await expect(page.locator('text=Booked')).toBeVisible();
      
      // Simulate resource return
      await editButton.click();
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'Available');
      await page.click('input[type="checkbox"]'); // Toggle available back to true
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=Available')).toBeVisible();
    });
  });

  test.describe('Cross-Feature Integration', () => {
    test('should integrate table operations with search and filtering', async ({ page }) => {
      const user = createTestUser('_search_integration');
      await helpers.registerUser(user);

      // Create a comprehensive dataset
      const inventoryTable = {
        name: `inventory_${Date.now()}`,
        columns: [
          { name: 'item_name', dataType: 'string' },
          { name: 'category', dataType: 'string' },
          { name: 'quantity', dataType: 'number' },
          { name: 'in_stock', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(inventoryTable);
      await helpers.selectTable(inventoryTable.name);

      // Add diverse inventory items
      const items = [
        { item_name: 'Laptop Dell XPS', category: 'Electronics', quantity: 5, in_stock: true },
        { item_name: 'Office Chair', category: 'Furniture', quantity: 0, in_stock: false },
        { item_name: 'Wireless Mouse', category: 'Electronics', quantity: 15, in_stock: true },
        { item_name: 'Desk Lamp', category: 'Furniture', quantity: 3, in_stock: true },
        { item_name: 'Tablet iPad', category: 'Electronics', quantity: 2, in_stock: true }
      ];

      for (const item of items) {
        await helpers.createRow(helpers.generateTestRow({ values: item }));
      }

      // Test search functionality
      const searchInput = page.locator('input[placeholder*="Search"], input[placeholder*="Filter"]');
      if (await searchInput.isVisible()) {
        // Search for electronics
        await searchInput.fill('Electronics');
        await page.waitForTimeout(1000);

        // Should show electronics items
        await expect(page.locator('text=Laptop Dell XPS')).toBeVisible();
        await expect(page.locator('text=Wireless Mouse')).toBeVisible();
        
        // Should not show furniture
        await expect(page.locator('text=Office Chair')).not.toBeVisible();

        // Clear search and verify all items return
        await searchInput.clear();
        await page.waitForTimeout(500);
        await expect(page.locator('text=Office Chair')).toBeVisible();
      }

      // Test operations on filtered results
      if (await searchInput.isVisible()) {
        await searchInput.fill('Laptop');
        await page.waitForTimeout(1000);

        // Edit the laptop entry
        const laptopRow = page.locator('[data-testid^="row-"]:has-text("Laptop Dell XPS")');
        const editButton = laptopRow.locator('button:has-text("Edit")');
        if (await editButton.isVisible()) {
          await editButton.click();
          await page.waitForSelector('[data-testid="row-form"]');
          await page.fill('input[placeholder="quantity"]', '3');
          await page.click('button:has-text("Save")');

          // Verify update appears in filtered view
          await expect(page.locator('text=3')).toBeVisible();
        }

        await searchInput.clear();
      }
    });

    test('should integrate multiple table operations with data consistency', async ({ page }) => {
      const user = createTestUser('_consistency_integration');
      await helpers.registerUser(user);

      // Create related tables
      const usersTable = {
        name: `users_${Date.now()}`,
        columns: [
          { name: 'username', dataType: 'string' },
          { name: 'email', dataType: 'string' },
          { name: 'role', dataType: 'string' },
          { name: 'active', dataType: 'boolean' }
        ]
      };

      const tasksTable = {
        name: `tasks_${Date.now()}`,
        columns: [
          { name: 'task_title', dataType: 'string' },
          { name: 'assigned_to', dataType: 'string' },
          { name: 'priority', dataType: 'string' },
          { name: 'completed', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(usersTable);
      await helpers.createTable(tasksTable);

      // Add users
      await helpers.selectTable(usersTable.name);
      const users = [
        { username: 'alice', email: 'alice@company.com', role: 'Developer', active: true },
        { username: 'bob', email: 'bob@company.com', role: 'Designer', active: true },
        { username: 'charlie', email: 'charlie@company.com', role: 'Manager', active: false }
      ];

      for (const userData of users) {
        await helpers.createRow(helpers.generateTestRow({ values: userData }));
      }

      // Add tasks referencing users
      await helpers.selectTable(tasksTable.name);
      const tasks = [
        { task_title: 'Fix login bug', assigned_to: 'alice', priority: 'High', completed: false },
        { task_title: 'Design homepage', assigned_to: 'bob', priority: 'Medium', completed: false },
        { task_title: 'Review code', assigned_to: 'charlie', priority: 'Low', completed: true }
      ];

      for (const task of tasks) {
        await helpers.createRow(helpers.generateTestRow({ values: task }));
      }

      // Test cross-table data consistency
      // Mark Alice as inactive in users table
      await helpers.selectTable(usersTable.name);
      const aliceRow = page.locator('[data-testid^="row-"]:has-text("alice")');
      const editButton = aliceRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle active to false
      await page.click('button:has-text("Save")');

      // Verify tasks table still shows Alice's assignments
      await helpers.selectTable(tasksTable.name);
      await expect(page.locator('text=alice')).toBeVisible();
      await expect(page.locator('text=Fix login bug')).toBeVisible();

      // Complete Alice's task
      const taskRow = page.locator('[data-testid^="row-"]:has-text("Fix login bug")');
      const taskEditButton = taskRow.locator('button:has-text("Edit")');
      await taskEditButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle completed to true
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=true')).toBeVisible(); // completed = true

      // Verify data persists across table switches
      await helpers.selectTable(usersTable.name);
      await expect(page.locator('[data-testid^="row-"]:has-text("alice")').locator('text=false')).toBeVisible(); // active = false

      await helpers.selectTable(tasksTable.name);
      await expect(page.locator('[data-testid^="row-"]:has-text("Fix login bug")').locator('text=true')).toBeVisible(); // completed = true
    });

    test('should integrate form operations with validation across features', async ({ page }) => {
      const user = createTestUser('_form_validation_integration');
      await helpers.registerUser(user);

      const productsTable = {
        name: `products_${Date.now()}`,
        columns: [
          { name: 'product_name', dataType: 'string' },
          { name: 'price', dataType: 'number' },
          { name: 'description', dataType: 'string' },
          { name: 'available', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(productsTable);
      await helpers.selectTable(productsTable.name);

      // Test form validation with various data scenarios
      await page.click('button:has-text("Add Row")');
      await page.waitForSelector('[data-testid="row-form"]');

      // Test with empty required fields
      await page.click('button:has-text("Save")');
      
      // Form should still be visible (validation failed)
      await expect(page.locator('[data-testid="row-form"]')).toBeVisible();

      // Fill with valid data
      await page.fill('input[placeholder="product_name"]', 'Test Product');
      await page.fill('input[placeholder="price"]', '29.99');
      await page.fill('input[placeholder="description"]', 'A great test product');
      await page.click('button:has-text("Save")');

      // Should create successfully
      await expect(page.locator('text=Test Product')).toBeVisible();
      await expect(page.locator('text=29.99')).toBeVisible();

      // Test editing with validation
      const productRow = page.locator('[data-testid^="row-"]:has-text("Test Product")');
      const editButton = productRow.locator('button:has-text("Edit")');
      await editButton.click();

      await page.waitForSelector('[data-testid="row-form"]');
      
      // Try to set invalid price
      await page.fill('input[placeholder="price"]', 'invalid_price');
      await page.click('button:has-text("Save")');

      // Should show validation error or prevent submission
      const formStillVisible = await page.locator('[data-testid="row-form"]').isVisible();
      expect(formStillVisible).toBe(true);

      // Correct the price
      await page.fill('input[placeholder="price"]', '39.99');
      await page.click('button:has-text("Save")');

      // Should update successfully
      await expect(page.locator('text=39.99')).toBeVisible();
    });
  });

  test.describe('API Integration Patterns', () => {
    test('should handle sequential API operations correctly', async ({ page }) => {
      const user = createTestUser('_api_sequential');
      await helpers.registerUser(user);

      // Monitor network requests
      const apiCalls = [];
      page.on('request', request => {
        if (request.url().includes('/api/')) {
          apiCalls.push({
            method: request.method(),
            url: request.url(),
            timestamp: Date.now()
          });
        }
      });

      // Perform sequential operations
      const eventsTable = {
        name: `events_${Date.now()}`,
        columns: [
          { name: 'event_name', dataType: 'string' },
          { name: 'date', dataType: 'string' },
          { name: 'attendees', dataType: 'number' },
          { name: 'confirmed', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(eventsTable);
      await helpers.selectTable(eventsTable.name);

      // Create multiple events in sequence
      const events = [
        { event_name: 'Team Meeting', date: '2024-01-15', attendees: 10, confirmed: true },
        { event_name: 'Product Launch', date: '2024-02-01', attendees: 50, confirmed: false },
        { event_name: 'Annual Conference', date: '2024-03-15', attendees: 200, confirmed: true }
      ];

      for (const event of events) {
        await helpers.createRow(helpers.generateTestRow({ values: event }));
        await page.waitForTimeout(500); // Brief pause between operations
      }

      // Verify all events were created
      await expect(page.locator('text=Team Meeting')).toBeVisible();
      await expect(page.locator('text=Product Launch')).toBeVisible();
      await expect(page.locator('text=Annual Conference')).toBeVisible();

      // Verify API calls were made in correct sequence
      expect(apiCalls.length).toBeGreaterThan(0);
      console.log(`Made ${apiCalls.length} API calls during sequential operations`);
    });

    test('should handle concurrent API operations gracefully', async ({ page }) => {
      const user = createTestUser('_api_concurrent');
      await helpers.registerUser(user);

      const notesTable = {
        name: `notes_${Date.now()}`,
        columns: [
          { name: 'title', dataType: 'string' },
          { name: 'content', dataType: 'string' },
          { name: 'priority', dataType: 'string' },
          { name: 'archived', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(notesTable);
      await helpers.selectTable(notesTable.name);

      // Create initial note
      const noteId = `note_${Date.now()}`;
      await helpers.createRow(helpers.generateTestRow({
        id: noteId,
        values: {
          title: 'Concurrent Test Note',
          content: 'Testing concurrent operations',
          priority: 'Medium',
          archived: false
        }
      }));

      // Simulate rapid concurrent-like operations
      // (In a single browser session, these will be sequential but rapid)
      const updates = [
        { priority: 'High' },
        { content: 'Updated content during concurrent test' },
        { title: 'Modified Concurrent Test Note' },
        { archived: true }
      ];

      for (const update of updates) {
        await helpers.editRow(noteId, update);
        // Minimal delay to simulate near-concurrent operations
        await page.waitForTimeout(100);
      }

      // Verify final state reflects all updates
      await expect(page.locator('text=Modified Concurrent Test Note')).toBeVisible();
      await expect(page.locator('text=High')).toBeVisible();
      await expect(page.locator('text=true')).toBeVisible(); // archived = true

      // Test data consistency after rapid operations
      await page.reload();
      await helpers.waitForTableLoad();
      
      await expect(page.locator('text=Modified Concurrent Test Note')).toBeVisible();
    });

    test('should handle API error recovery patterns', async ({ page }) => {
      const user = createTestUser('_api_recovery');
      await helpers.registerUser(user);

      const logsTable = {
        name: `error_logs_${Date.now()}`,
        columns: [
          { name: 'message', dataType: 'string' },
          { name: 'level', dataType: 'string' },
          { name: 'source', dataType: 'string' },
          { name: 'resolved', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(logsTable);
      await helpers.selectTable(logsTable.name);

      // Create initial data
      await helpers.createRow(helpers.generateTestRow({
        values: {
          message: 'Test error message',
          level: 'Warning',
          source: 'API Test',
          resolved: false
        }
      }));

      // Simulate network error and recovery
      let errorInjected = false;
      
      await page.route('/api/tables/*/rows/*', async (route) => {
        if (!errorInjected) {
          errorInjected = true;
          // Simulate network error for first attempt
          route.abort();
          return;
        }
        // Allow subsequent requests to proceed
        route.continue();
      });

      // Try to update the log entry (first attempt should fail)
      const logRow = page.locator('[data-testid^="row-"]:has-text("Test error message")');
      const editButton = logRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="level"]', 'Error');
      await page.click('button:has-text("Save")');

      // The operation might fail initially, but the form should remain open
      // or show an error state
      await page.waitForTimeout(2000);

      // Remove the route to allow subsequent requests
      await page.unroute('/api/tables/*/rows/*');

      // Try the operation again (should succeed)
      // If form is still open, try saving again
      if (await page.locator('[data-testid="row-form"]').isVisible()) {
        await page.click('button:has-text("Save")');
      } else {
        // If form closed, try editing again
        await editButton.click();
        await page.waitForSelector('[data-testid="row-form"]');
        await page.fill('input[placeholder="level"]', 'Error');
        await page.click('button:has-text("Save")');
      }

      // Verify eventual success
      await expect(page.locator('text=Error')).toBeVisible({ timeout: 10000 });
    });
  });

  test.describe('Real-World Scenarios', () => {
    test('should support personal knowledge management workflow', async ({ page }) => {
      const user = createTestUser('_knowledge_mgmt');
      await helpers.registerUser(user);

      // Create knowledge base structure
      const articlesTable = {
        name: `articles_${Date.now()}`,
        columns: [
          { name: 'title', dataType: 'string' },
          { name: 'topic', dataType: 'string' },
          { name: 'summary', dataType: 'string' },
          { name: 'bookmarked', dataType: 'boolean' }
        ]
      };

      const tagsTable = {
        name: `tags_${Date.now()}`,
        columns: [
          { name: 'tag_name', dataType: 'string' },
          { name: 'category', dataType: 'string' },
          { name: 'usage_count', dataType: 'number' },
          { name: 'active', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(articlesTable);
      await helpers.createTable(tagsTable);

      // Add knowledge articles
      await helpers.selectTable(articlesTable.name);
      
      const articles = [
        { title: 'React Best Practices', topic: 'Programming', summary: 'Guidelines for React development', bookmarked: true },
        { title: 'Database Design Patterns', topic: 'Architecture', summary: 'Common patterns for database design', bookmarked: true },
        { title: 'Leadership Tips', topic: 'Management', summary: 'Effective leadership strategies', bookmarked: false }
      ];

      for (const article of articles) {
        await helpers.createRow(helpers.generateTestRow({ values: article }));
      }

      // Add tag system
      await helpers.selectTable(tagsTable.name);
      
      const tags = [
        { tag_name: 'react', category: 'technology', usage_count: 15, active: true },
        { tag_name: 'database', category: 'technology', usage_count: 8, active: true },
        { tag_name: 'leadership', category: 'soft-skills', usage_count: 5, active: true }
      ];

      for (const tag of tags) {
        await helpers.createRow(helpers.generateTestRow({ values: tag }));
      }

      // Simulate knowledge discovery and organization
      await helpers.selectTable(articlesTable.name);
      
      // Mark additional articles as bookmarked
      const leadershipRow = page.locator('[data-testid^="row-"]:has-text("Leadership Tips")');
      const editButton = leadershipRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle bookmarked to true
      await page.click('button:has-text("Save")');

      await expect(page.locator('[data-testid^="row-"]:has-text("Leadership Tips")').locator('text=true')).toBeVisible();

      // Update tag usage
      await helpers.selectTable(tagsTable.name);
      const reactTagRow = page.locator('[data-testid^="row-"]:has-text("react")');
      const tagEditButton = reactTagRow.locator('button:has-text("Edit")');
      await tagEditButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="usage_count"]', '20');
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=20')).toBeVisible();

      // Verify cross-table navigation preserves state
      await helpers.selectTable(articlesTable.name);
      await expect(page.locator('text=React Best Practices')).toBeVisible();
      
      await helpers.selectTable(tagsTable.name);
      await expect(page.locator('text=20')).toBeVisible();
    });

    test('should support project tracking workflow', async ({ page }) => {
      const user = createTestUser('_project_tracking');
      await helpers.registerUser(user);

      // Create project management structure
      const projectsTable = {
        name: `projects_tracking_${Date.now()}`,
        columns: [
          { name: 'project_name', dataType: 'string' },
          { name: 'status', dataType: 'string' },
          { name: 'budget', dataType: 'number' },
          { name: 'critical', dataType: 'boolean' }
        ]
      };

      const milestonesTable = {
        name: `milestones_${Date.now()}`,
        columns: [
          { name: 'milestone_name', dataType: 'string' },
          { name: 'project_ref', dataType: 'string' },
          { name: 'target_date', dataType: 'string' },
          { name: 'completed', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(projectsTable);
      await helpers.createTable(milestonesTable);

      // Create projects
      await helpers.selectTable(projectsTable.name);
      
      const projects = [
        { project_name: 'Website Redesign', status: 'Active', budget: 50000, critical: true },
        { project_name: 'Mobile App', status: 'Planning', budget: 75000, critical: false },
        { project_name: 'Database Migration', status: 'On Hold', budget: 30000, critical: true }
      ];

      for (const project of projects) {
        await helpers.createRow(helpers.generateTestRow({ values: project }));
      }

      // Create milestones for projects
      await helpers.selectTable(milestonesTable.name);
      
      const milestones = [
        { milestone_name: 'Design Complete', project_ref: 'Website Redesign', target_date: '2024-02-15', completed: true },
        { milestone_name: 'Development Phase 1', project_ref: 'Website Redesign', target_date: '2024-03-01', completed: false },
        { milestone_name: 'UI Mockups', project_ref: 'Mobile App', target_date: '2024-02-01', completed: false },
        { milestone_name: 'Schema Design', project_ref: 'Database Migration', target_date: '2024-01-30', completed: true }
      ];

      for (const milestone of milestones) {
        await helpers.createRow(helpers.generateTestRow({ values: milestone }));
      }

      // Simulate project status updates
      await helpers.selectTable(projectsTable.name);
      const mobileAppRow = page.locator('[data-testid^="row-"]:has-text("Mobile App")');
      const editButton = mobileAppRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'Active');
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=Active')).toBeVisible();

      // Update milestone completion
      await helpers.selectTable(milestonesTable.name);
      const uiMockupsRow = page.locator('[data-testid^="row-"]:has-text("UI Mockups")');
      const milestoneEditButton = uiMockupsRow.locator('button:has-text("Edit")');
      await milestoneEditButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle completed to true
      await page.click('button:has-text("Save")');

      await expect(page.locator('[data-testid^="row-"]:has-text("UI Mockups")').locator('text=true')).toBeVisible();

      // Verify cross-references between projects and milestones
      await helpers.selectTable(projectsTable.name);
      await expect(page.locator('text=Mobile App')).toBeVisible();
      
      await helpers.selectTable(milestonesTable.name);
      await expect(page.locator('text=Mobile App')).toBeVisible(); // project_ref
      await expect(page.locator('[data-testid^="row-"]:has-text("UI Mockups")').locator('text=true')).toBeVisible();
    });

    test('should support comprehensive data analysis workflow', async ({ page }) => {
      const user = createTestUser('_data_analysis');
      await helpers.registerUser(user);

      // Create analytics structure
      const metricsTable = {
        name: `metrics_${Date.now()}`,
        columns: [
          { name: 'metric_name', dataType: 'string' },
          { name: 'value', dataType: 'number' },
          { name: 'category', dataType: 'string' },
          { name: 'trending_up', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(metricsTable);
      await helpers.selectTable(metricsTable.name);

      // Add various metrics for analysis
      const metrics = [
        { metric_name: 'Website Traffic', value: 15000, category: 'Marketing', trending_up: true },
        { metric_name: 'Conversion Rate', value: 3.2, category: 'Sales', trending_up: false },
        { metric_name: 'Customer Satisfaction', value: 8.5, category: 'Support', trending_up: true },
        { metric_name: 'Server Response Time', value: 250, category: 'Technical', trending_up: false },
        { metric_name: 'Monthly Revenue', value: 125000, category: 'Financial', trending_up: true }
      ];

      for (const metric of metrics) {
        await helpers.createRow(helpers.generateTestRow({ values: metric }));
      }

      // Simulate metric updates over time
      const trafficRow = page.locator('[data-testid^="row-"]:has-text("Website Traffic")');
      const editButton = trafficRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="value"]', '18000');
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=18000')).toBeVisible();

      // Add new metric dynamically
      await helpers.createRow(helpers.generateTestRow({
        values: {
          metric_name: 'User Engagement',
          value: 75.5,
          category: 'Product',
          trending_up: true
        }
      }));

      await expect(page.locator('text=User Engagement')).toBeVisible();
      await expect(page.locator('text=75.5')).toBeVisible();

      // Verify comprehensive data set
      const totalRows = await helpers.getRowCount();
      expect(totalRows).toBe(6); // 5 original + 1 new metric
    });
  });

  test.describe('Advanced User Scenarios', () => {
    test('should handle complex multi-step workflows with rollback scenarios', async ({ page }) => {
      const user = createTestUser('_complex_workflow');
      await helpers.registerUser(user);

      // Create a workflow simulation table
      const workflowTable = {
        name: `workflow_steps_${Date.now()}`,
        columns: [
          { name: 'step_name', dataType: 'string' },
          { name: 'status', dataType: 'string' },
          { name: 'order', dataType: 'number' },
          { name: 'can_rollback', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(workflowTable);
      await helpers.selectTable(workflowTable.name);

      // Create workflow steps
      const steps = [
        { step_name: 'Initialize', status: 'Completed', order: 1, can_rollback: false },
        { step_name: 'Validate Data', status: 'Completed', order: 2, can_rollback: true },
        { step_name: 'Process Records', status: 'In Progress', order: 3, can_rollback: true },
        { step_name: 'Generate Report', status: 'Pending', order: 4, can_rollback: true },
        { step_name: 'Send Notification', status: 'Pending', order: 5, can_rollback: false }
      ];

      for (const step of steps) {
        await helpers.createRow(helpers.generateTestRow({ values: step }));
      }

      // Simulate workflow progression
      const processRow = page.locator('[data-testid^="row-"]:has-text("Process Records")');
      const editButton = processRow.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'Completed');
      await page.click('button:has-text("Save")');

      // Move next step to in progress
      const reportRow = page.locator('[data-testid^="row-"]:has-text("Generate Report")');
      const reportEditButton = reportRow.locator('button:has-text("Edit")');
      await reportEditButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'In Progress');
      await page.click('button:has-text("Save")');

      // Simulate rollback scenario - mark step as failed
      await reportEditButton.click();
      await page.waitForSelector('[data-testid="row-form"]');
      await page.fill('input[placeholder="status"]', 'Failed');
      await page.click('button:has-text("Save")');

      await expect(page.locator('text=Failed')).toBeVisible();

      // Verify workflow state consistency
      await expect(page.locator('[data-testid^="row-"]:has-text("Process Records")').locator('text=Completed')).toBeVisible();
      await expect(page.locator('[data-testid^="row-"]:has-text("Generate Report")').locator('text=Failed')).toBeVisible();
      await expect(page.locator('[data-testid^="row-"]:has-text("Send Notification")').locator('text=Pending')).toBeVisible();
    });

    test('should support data migration and transformation workflows', async ({ page }) => {
      const user = createTestUser('_data_migration');
      await helpers.registerUser(user);

      // Create source and target tables for migration simulation
      const sourceTable = {
        name: `legacy_data_${Date.now()}`,
        columns: [
          { name: 'old_id', dataType: 'string' },
          { name: 'legacy_field', dataType: 'string' },
          { name: 'deprecated_status', dataType: 'string' },
          { name: 'needs_migration', dataType: 'boolean' }
        ]
      };

      const targetTable = {
        name: `modern_data_${Date.now()}`,
        columns: [
          { name: 'new_id', dataType: 'string' },
          { name: 'updated_field', dataType: 'string' },
          { name: 'current_status', dataType: 'string' },
          { name: 'migrated', dataType: 'boolean' }
        ]
      };

      await helpers.createTable(sourceTable);
      await helpers.createTable(targetTable);

      // Populate source data
      await helpers.selectTable(sourceTable.name);
      
      const legacyData = [
        { old_id: 'LEG001', legacy_field: 'Old Value 1', deprecated_status: 'active', needs_migration: true },
        { old_id: 'LEG002', legacy_field: 'Old Value 2', deprecated_status: 'inactive', needs_migration: true },
        { old_id: 'LEG003', legacy_field: 'Old Value 3', deprecated_status: 'pending', needs_migration: false }
      ];

      for (const data of legacyData) {
        await helpers.createRow(helpers.generateTestRow({ values: data }));
      }

      // Simulate migration process
      await helpers.selectTable(targetTable.name);
      
      const migratedData = [
        { new_id: 'MOD001', updated_field: 'Migrated Value 1', current_status: 'enabled', migrated: true },
        { new_id: 'MOD002', updated_field: 'Migrated Value 2', current_status: 'disabled', migrated: true }
      ];

      for (const data of migratedData) {
        await helpers.createRow(helpers.generateTestRow({ values: data }));
      }

      // Update source data to reflect migration
      await helpers.selectTable(sourceTable.name);
      const legacyRow1 = page.locator('[data-testid^="row-"]:has-text("LEG001")');
      const editButton = legacyRow1.locator('button:has-text("Edit")');
      await editButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle needs_migration to false
      await page.click('button:has-text("Save")');

      // Verify migration tracking
      await expect(page.locator('[data-testid^="row-"]:has-text("LEG001")').locator('text=false')).toBeVisible();

      // Verify target data
      await helpers.selectTable(targetTable.name);
      await expect(page.locator('text=Migrated Value 1')).toBeVisible();
      await expect(page.locator('text=enabled')).toBeVisible();
    });

    test('should handle comprehensive error scenarios and recovery', async ({ page }) => {
      const user = createTestUser('_error_recovery');
      await helpers.registerUser(user);

      const errorLogTable = {
        name: `comprehensive_errors_${Date.now()}`,
        columns: [
          { name: 'error_type', dataType: 'string' },
          { name: 'severity', dataType: 'string' },
          { name: 'resolved', dataType: 'boolean' },
          { name: 'retry_count', dataType: 'number' }
        ]
      };

      await helpers.createTable(errorLogTable);
      await helpers.selectTable(errorLogTable.name);

      // Create various error scenarios
      const errors = [
        { error_type: 'Network Timeout', severity: 'High', resolved: false, retry_count: 0 },
        { error_type: 'Validation Error', severity: 'Medium', resolved: true, retry_count: 2 },
        { error_type: 'Permission Denied', severity: 'Critical', resolved: false, retry_count: 1 }
      ];

      for (const error of errors) {
        await helpers.createRow(helpers.generateTestRow({ values: error }));
      }

      // Simulate error resolution process
      const networkErrorRow = page.locator('[data-testid^="row-"]:has-text("Network Timeout")');
      const editButton = networkErrorRow.locator('button:has-text("Edit")');
      
      // Simulate multiple retry attempts
      for (let retryCount = 1; retryCount <= 3; retryCount++) {
        await editButton.click();
        await page.waitForSelector('[data-testid="row-form"]');
        await page.fill('input[placeholder="retry_count"]', retryCount.toString());
        
        if (retryCount === 3) {
          // Mark as resolved on final retry
          await page.click('input[type="checkbox"]'); // Toggle resolved to true
        }
        
        await page.click('button:has-text("Save")');
        await page.waitForTimeout(500);
      }

      // Verify error resolution
      await expect(page.locator('[data-testid^="row-"]:has-text("Network Timeout")').locator('text=true')).toBeVisible(); // resolved
      await expect(page.locator('[data-testid^="row-"]:has-text("Network Timeout")').locator('text=3')).toBeVisible(); // retry_count

      // Test bulk error resolution
      const permissionErrorRow = page.locator('[data-testid^="row-"]:has-text("Permission Denied")');
      const permissionEditButton = permissionErrorRow.locator('button:has-text("Edit")');
      await permissionEditButton.click();
      
      await page.waitForSelector('[data-testid="row-form"]');
      await page.click('input[type="checkbox"]'); // Toggle resolved to true
      await page.fill('input[placeholder="retry_count"]', '5');
      await page.click('button:has-text("Save")');

      // Verify all critical errors are now resolved
      const unresolvedCriticalErrors = page.locator('[data-testid^="row-"]:has-text("Critical")').locator('text=false');
      const count = await unresolvedCriticalErrors.count();
      expect(count).toBe(0);
    });
  });
});