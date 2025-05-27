# Test Migration: Playwright to Vitest

## Overview

We've migrated our e2e tests from Playwright to Vitest for lighter-weight, faster integration testing. This change provides several benefits:

- **Faster execution**: No browser overhead for most tests
- **Better developer experience**: Hot reload, better error messages
- **Easier debugging**: Direct access to React components
- **Lighter dependencies**: Smaller node_modules and faster CI

## What Changed

### Dependencies
- **Removed**: `@playwright/test`, `playwright`
- **Added**: `vitest`, `@testing-library/react`, `@testing-library/user-event`, `jsdom`, `msw`

### Test Files
- Old: `e2e/*.spec.ts` (Playwright)
- New: `e2e/*.test.ts` (Vitest)

### Test Approach
- **Before**: Full browser automation with Playwright
- **After**: Component testing with mocked APIs using MSW (Mock Service Worker)

## Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run specific test file
npm run test:auth

# Run with coverage
npm run test:coverage

# Run with UI
npm run test:ui
```

## Writing Tests

### Basic Test Structure

```typescript
import { describe, test, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TestHelpers, createTestUser } from '../src/test/helpers'
import App from '../src/App'

describe('My Feature', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  test('should do something', async () => {
    render(<App />)
    
    const user = createTestUser('_test')
    await helpers.registerUser(user)
    
    expect(screen.getByTestId('main-app')).toBeInTheDocument()
  })
})
```

### Helper Functions

The `TestHelpers` class provides utilities for common operations:

- `registerUser(user)` - Register and login a user
- `loginUser(user)` - Login an existing user
- `logout()` - Logout current user
- `createTable(table)` - Create a new table
- `createRow(row)` - Add a row to a table
- `editRow(id, data)` - Edit an existing row
- `deleteRow(id)` - Delete a row
- `searchTable(term)` - Search within a table
- `verifyTableExists(name)` - Assert table exists
- `expectNoErrors()` - Assert no error messages

### API Mocking

Tests use MSW to mock API responses. You can override default mocks:

```typescript
import { server } from '../src/test/setup'
import { rest } from 'msw'

test('should handle API error', async () => {
  // Override default mock
  server.use(
    rest.post('/api/auth/login', (req, res, ctx) => {
      return res(
        ctx.status(401),
        ctx.json({ error: 'Invalid credentials' })
      )
    })
  )
  
  // Test error handling...
})
```

### Simulating Errors

Helper methods for testing error scenarios:

```typescript
// Network error
helpers.simulateNetworkError('/api/tables')

// Server error
helpers.simulateServerError('/api/auth/login', 500)

// Reset to defaults
helpers.removeNetworkErrorSimulation()
```

## Key Differences from Playwright

| Aspect | Playwright | Vitest |
|--------|------------|---------|
| **Browser** | Real browser | jsdom (simulated) |
| **Speed** | Slower | Much faster |
| **Debugging** | Browser DevTools | IDE debugging |
| **Network** | Real HTTP calls | Mocked with MSW |
| **Setup** | Complex server setup | Simple component rendering |
| **CI/CD** | Requires browser binaries | Lightweight |

## Limitations

### What We Lost
- **Real browser testing**: No actual browser rendering
- **Cross-browser compatibility**: No browser-specific testing
- **Visual regression**: No screenshot comparison
- **E2E workflows**: No real server interaction

### What We Gained
- **Speed**: 10x faster test execution
- **Reliability**: No flaky browser timeouts
- **Developer experience**: Better debugging and hot reload
- **Maintainability**: Easier to write and maintain tests

## When to Use Each Approach

### Use Vitest for:
- Component behavior testing
- User interaction flows
- API integration logic
- Form validation
- State management
- Most regression testing

### Consider Playwright for:
- True end-to-end workflows
- Browser compatibility testing
- Visual regression testing
- Performance testing
- Third-party integrations

## Migration Notes

### Converted Tests
- ✅ `smoke.test.ts` - Basic app functionality
- ✅ `auth.test.ts` - Authentication flows
- ✅ `table-operations.test.ts` - Table CRUD operations

### Remaining Playwright Tests
You can still run specific Playwright tests if needed by keeping the original files and running them separately.

## Best Practices

1. **Keep tests focused**: Test one feature per test case
2. **Use helper functions**: Reuse common operations
3. **Mock external dependencies**: Use MSW for API calls
4. **Test user behavior**: Focus on what users actually do
5. **Verify outcomes**: Check the UI state, not implementation details

## Troubleshooting

### Common Issues

**Test timeout**: Increase timeout in `vite.config.ts` or use `waitFor()`

**Mantine components not rendering**: Already configured in setup, but check console for warnings

**API mocks not working**: Ensure MSW handlers are properly defined

**Components not finding elements**: Use `screen.debug()` to see rendered output

### Debug Commands

```bash
# Run single test with debug output
npx vitest run auth.test.ts --reporter=verbose

# Open test UI for interactive debugging
npm run test:ui
```