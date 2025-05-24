# Playwright E2E Tests

This directory contains end-to-end tests for the Notably frontend application using Playwright.

## Setup

### Prerequisites

1. Ensure you have Node.js installed
2. Install dependencies: `npm install`
3. Install Playwright browsers: `npm run test:install`

### Quick Start

```bash
# Install Playwright browsers (one-time setup)
npm run test:install

# Run all tests in headless mode
npm run test

# Run tests with browser UI visible
npm run test:headed

# Run tests with Playwright UI for debugging
npm run test:ui
```

## Available Test Scripts

| Script | Description |
|--------|-------------|
| `npm run test` | Run all tests in headless mode with list reporter |
| `npm run test:headless` | Same as `test` - explicit headless mode |
| `npm run test:headed` | Run tests with browser UI visible |
| `npm run test:ui` | Open Playwright UI for interactive test running |
| `npm run test:debug` | Run tests in debug mode with step-by-step execution |
| `npm run test:report` | Open the HTML test report |
| `npm run test:no-server` | Run tests without auto-starting servers |
| `npm run test:auth` | Run only authentication tests |
| `npm run test:tables` | Run only table operation tests |
| `npm run test:history` | Run only history/snapshot tests |
| `npm run test:regression` | Run only regression tests |

## Test Organization

### Test Files

- **`auth.spec.ts`** - User authentication (login, register, logout, session management)
- **`table-operations.spec.ts`** - Table and row CRUD operations
- **`history-snapshot.spec.ts`** - Historical data and snapshot features
- **`regression.spec.ts`** - Regression tests for known bugs
- **`smoke.spec.ts`** - Basic smoke tests to verify Playwright setup

### Test Helpers

- **`helpers/test-helpers.ts`** - Reusable test utilities and page object methods

## Running Tests

### Development Workflow

1. **Start servers manually** (recommended for development):
   ```bash
   # Terminal 1: Start frontend
   npm run dev
   
   # Terminal 2: Start backend
   npm run start:backend
   
   # Terminal 3: Run tests
   npm run test:no-server
   ```

2. **Start servers automatically** (slower but convenient):
   ```bash
   npm run test  # Will auto-start frontend and backend
   ```

3. **Start both servers together**:
   ```bash
   npm run dev:all  # Uses concurrently to start both servers
   ```

### Test Modes

#### Headless Mode (Default)
```bash
npm run test
```
- Runs without visible browser windows
- Faster execution
- Good for CI/CD and regression testing

#### Headed Mode
```bash
npm run test:headed
```
- Shows browser windows during test execution
- Useful for debugging and development
- Slower but visual feedback

#### Debug Mode
```bash
npm run test:debug
```
- Runs one test at a time with debugging capabilities
- Opens browser dev tools
- Allows step-by-step execution

#### UI Mode
```bash
npm run test:ui
```
- Opens Playwright's interactive UI
- Best for test development and debugging
- Visual test runner with timeline and trace viewer

## Test Configuration

### Main Config (`playwright.config.ts`)
- Full configuration with auto-starting web servers
- Multiple browser projects (Chrome, Firefox, Safari)
- Comprehensive reporting

### No-Server Config (`playwright.config.no-server.ts`)
- Simplified configuration without auto web server startup
- Faster test startup when servers are already running
- Single browser project (Chrome) for speed

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HEADED` | Run tests in headed mode | `false` |
| `CI` | Enable CI-specific settings | `false` |

## Test Data and Cleanup

### Test Users and Data
- Tests automatically generate unique test data using timestamps
- Each test creates isolated test users and tables
- Test data prefixes: `testuser`, `test_table_`, etc.

### Cleanup
- Tests are designed to be independent
- Unique naming prevents data conflicts
- Manual cleanup may be needed for development databases

## Troubleshooting

### Common Issues

1. **Tests hang or timeout**
   - Ensure frontend (localhost:5173) and backend (localhost:8080) are running
   - Check if Playwright browsers are installed: `npm run test:install`
   - Use `npm run test:no-server` if servers are already running

2. **Browser not found errors**
   ```bash
   npm run test:install
   ```

3. **TypeScript errors**
   - Ensure `@types/node` is installed
   - Check `tsconfig.node.json` includes Playwright config

4. **Network timeouts**
   - Increase timeout values in config
   - Check server health endpoints
   - Ensure no firewall blocking localhost ports

### Debug Tips

1. **Use headed mode for visual debugging**:
   ```bash
   npm run test:headed -- auth.spec.ts
   ```

2. **Use debug mode for step-by-step execution**:
   ```bash
   npm run test:debug -- auth.spec.ts
   ```

3. **Check test reports**:
   ```bash
   npm run test:report
   ```

4. **Run specific tests**:
   ```bash
   npm run test -- --grep "should register a new user"
   ```

## Writing New Tests

### Test Structure
```typescript
import { test, expect } from '@playwright/test';
import { TestHelpers, createTestUser } from './helpers/test-helpers';

test.describe('Feature Name', () => {
  let helpers: TestHelpers;

  test.beforeEach(async ({ page }) => {
    helpers = new TestHelpers(page);
  });

  test('should do something', async ({ page }) => {
    // Test implementation
  });
});
```

### Best Practices
1. Use data-testid attributes for reliable element selection
2. Create unique test data for each test
3. Use the TestHelpers class for common operations
4. Add proper wait conditions for dynamic content
5. Clean up test data when possible
6. Use descriptive test names and organize with describe blocks

## CI/CD Integration

For continuous integration, use:
```bash
npm run test  # Runs with auto-server startup and retry logic
```

The configuration automatically adjusts for CI environments:
- Reduces parallelism
- Increases retry attempts
- Uses appropriate reporters