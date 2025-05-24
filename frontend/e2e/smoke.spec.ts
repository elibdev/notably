import { test, expect } from '@playwright/test';

test.describe('Smoke Tests', () => {
  test('Playwright setup is working', async ({ page }) => {
    // Test basic Playwright functionality without requiring servers
    await page.setContent(`
      <html>
        <head><title>Smoke Test</title></head>
        <body>
          <h1 data-testid="title">Playwright is working!</h1>
          <button data-testid="test-button">Click me</button>
          <p data-testid="result" style="display: none;">Button clicked!</p>
          <script>
            document.querySelector('[data-testid="test-button"]').addEventListener('click', () => {
              document.querySelector('[data-testid="result"]').style.display = 'block';
            });
          </script>
        </body>
      </html>
    `);

    // Verify page loaded correctly
    await expect(page.locator('[data-testid="title"]')).toHaveText('Playwright is working!');
    
    // Test interaction
    await page.click('[data-testid="test-button"]');
    await expect(page.locator('[data-testid="result"]')).toBeVisible();
    
    // Test page title
    await expect(page).toHaveTitle('Smoke Test');
  });

  test('Browser capabilities work correctly', async ({ page, browserName }) => {
    // Test browser-specific functionality
    await page.setContent(`
      <html>
        <body>
          <div data-testid="browser-info">Browser: ${browserName}</div>
          <input data-testid="text-input" placeholder="Type here" />
          <div data-testid="output"></div>
          <script>
            document.querySelector('[data-testid="text-input"]').addEventListener('input', (e) => {
              document.querySelector('[data-testid="output"]').textContent = e.target.value;
            });
          </script>
        </body>
      </html>
    `);

    // Test text input
    await page.fill('[data-testid="text-input"]', 'Hello Playwright!');
    await expect(page.locator('[data-testid="output"]')).toHaveText('Hello Playwright!');
    
    // Verify browser info is displayed
    await expect(page.locator('[data-testid="browser-info"]')).toContainText('Browser:');
  });

  test('Screenshots and basic assertions work', async ({ page }) => {
    await page.setContent(`
      <html>
        <head>
          <style>
            .test-box { 
              width: 100px; 
              height: 100px; 
              background-color: #007acc; 
              margin: 20px; 
            }
          </style>
        </head>
        <body>
          <div class="test-box" data-testid="blue-box"></div>
          <span data-testid="timestamp">${new Date().toISOString()}</span>
        </body>
      </html>
    `);

    // Test element visibility and properties
    const blueBox = page.locator('[data-testid="blue-box"]');
    await expect(blueBox).toBeVisible();
    await expect(blueBox).toHaveCSS('background-color', 'rgb(0, 122, 204)');
    
    // Test screenshot capability (this validates the test environment)
    await page.screenshot({ path: 'test-results/smoke-test.png' });
    
    // Test timestamp element exists
    await expect(page.locator('[data-testid="timestamp"]')).toBeVisible();
  });
});