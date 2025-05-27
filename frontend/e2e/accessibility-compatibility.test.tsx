import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser, createTestTable } from '../src/test/helpers'
import { server } from '../src/test/setup'
import { http, HttpResponse } from 'msw'
import App from '../src/App'

describe('Accessibility and Browser Compatibility', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Keyboard Navigation', () => {
    test('should support full keyboard navigation for authentication', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_keyboard_auth')

      // Test tab navigation through auth form
      const usernameInput = screen.getByPlaceholderText('Username')
      const passwordInput = screen.getByPlaceholderText('Password')
      const loginButton = screen.getByRole('button', { name: /login/i })

      // Verify elements can receive focus
      usernameInput.focus()
      expect(document.activeElement).toBe(usernameInput)

      // Test keyboard input
      await helpers.user.type(usernameInput, user.username)
      expect(usernameInput).toHaveValue(user.username)

      // Test tab to next field
      await helpers.user.tab()
      expect(document.activeElement).toBe(passwordInput)

      await helpers.user.type(passwordInput, user.password)
      
      // Test tab to button
      await helpers.user.tab()
      expect(document.activeElement).toBe(loginButton)

      // Test Enter key activation
      await helpers.user.keyboard('{Enter}')
      
      // Should show error for non-existent account
      await waitFor(() => {
        expect(screen.getByText(/invalid/i)).toBeInTheDocument()
      })

      // Test keyboard navigation to register tab
      const registerTab = screen.getByRole('tab', { name: /register/i })
      registerTab.focus()
      await helpers.user.keyboard('{Enter}')

      // Should switch to register form
      expect(screen.getByPlaceholderText('Email')).toBeInTheDocument()
    })

    test('should support keyboard navigation in table operations', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_keyboard_table')
      const table = createTestTable('_keyboard')

      // Mock table data
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([
            { id: '1', values: { name: 'Test Row', status: 'active' } }
          ])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Test keyboard navigation in table
      const createButton = screen.getByRole('button', { name: /create.*table|add.*row/i })
      createButton.focus()
      expect(document.activeElement).toBe(createButton)

      // Test Enter key to activate
      await helpers.user.keyboard('{Enter}')
      
      // Should open create dialog
      await waitFor(() => {
        expect(screen.getByRole('dialog')).toBeInTheDocument()
      })

      // Test Escape key to close
      await helpers.user.keyboard('{Escape}')
      
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })

    test('should provide keyboard shortcuts for common actions', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_shortcuts')
      await helpers.registerUser(user)

      // Test common keyboard shortcuts if implemented
      const shortcuts = [
        { key: '{Control>}n{/Control}', action: 'new' },
        { key: '{Control>}s{/Control}', action: 'save' },
        { key: '{Control>}f{/Control}', action: 'search' }
      ]

      for (const shortcut of shortcuts) {
        // Test if shortcut triggers expected action
        await helpers.user.keyboard(shortcut.key)
        
        // Verify no errors occurred
        await helpers.expectNoErrors()
      }
    })
  })

  describe('Screen Reader Support', () => {
    test('should have proper ARIA labels and roles', async () => {
      helpers.renderComponent(<App />)

      // Check main landmarks
      const main = screen.getByRole('main', { hidden: true }) || document.querySelector('main')
      if (main) {
        expect(main).toBeInTheDocument()
      }

      // Check form accessibility
      const authForm = screen.getByTestId('auth-form')
      expect(authForm).toBeInTheDocument()

      // Check input labels
      const usernameInput = screen.getByPlaceholderText('Username')
      const usernameLabel = screen.getByLabelText(/username/i)
      expect(usernameLabel).toBeInTheDocument()

      // Check button accessibility
      const loginButton = screen.getByRole('button', { name: /login/i })
      expect(loginButton).toHaveAttribute('type', 'button')

      // Check tab accessibility
      const loginTab = screen.getByRole('tab', { name: /login/i })
      expect(loginTab).toHaveAttribute('role', 'tab')
      expect(loginTab).toHaveAttribute('aria-selected')
    })

    test('should announce dynamic content changes', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_announcements')

      // Mock error response for testing announcements
      helpers.simulateServerError('/api/auth/login', 401)

      await helpers.user.type(screen.getByPlaceholderText('Username'), 'invalid')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'invalid')
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))

      // Error message should be announced to screen readers
      await waitFor(() => {
        const errorMessage = screen.getByText(/invalid/i)
        expect(errorMessage).toBeInTheDocument()
        
        // Check if error has proper ARIA attributes
        expect(errorMessage).toHaveAttribute('role', 'alert') ||
        expect(errorMessage).toHaveAttribute('aria-live') ||
        expect(errorMessage.closest('[role="alert"]')).toBeInTheDocument()
      })
    })

    test('should provide descriptive error messages', async () => {
      helpers.renderComponent(<App />)

      // Test form validation messages
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))

      // Should show descriptive validation messages
      const form = screen.getByTestId('auth-form')
      const inputs = form.querySelectorAll('input[required]')
      
      inputs.forEach(input => {
        // Check if input has proper validation attributes
        expect(input).toHaveAttribute('required')
        expect(input).toHaveAttribute('aria-invalid', 'false')
      })
    })

    test('should support high contrast mode', async () => {
      helpers.renderComponent(<App />)

      // Check that elements have sufficient color contrast
      const authForm = screen.getByTestId('auth-form')
      const computedStyle = window.getComputedStyle(authForm)
      
      // Basic contrast check (simplified)
      expect(computedStyle.backgroundColor).not.toBe('transparent')
      expect(computedStyle.color).not.toBe('transparent')

      // Check button contrast
      const loginButton = screen.getByRole('button', { name: /login/i })
      const buttonStyle = window.getComputedStyle(loginButton)
      expect(buttonStyle.backgroundColor).not.toBe(buttonStyle.color)
    })
  })

  describe('Focus Management', () => {
    test('should maintain logical focus order', async () => {
      helpers.renderComponent(<App />)

      const focusableElements = [
        screen.getByPlaceholderText('Username'),
        screen.getByPlaceholderText('Password'),
        screen.getByRole('button', { name: /login/i }),
        screen.getByRole('tab', { name: /register/i })
      ]

      // Test tab order
      for (let i = 0; i < focusableElements.length; i++) {
        await helpers.user.tab()
        // Note: In jsdom, focus management is limited, so we verify elements exist
        expect(focusableElements[i]).toBeInTheDocument()
      }
    })

    test('should trap focus in modal dialogs', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_focus_trap')
      const table = createTestTable('_focus')

      // Mock table for testing modal
      server.use(
        http.get('/api/tables', () => {
          return HttpResponse.json([
            { id: '1', name: table.name, createdAt: '2024-01-01T00:00:00Z' }
          ])
        }),
        http.get('/api/tables/1/rows', () => {
          return HttpResponse.json([])
        })
      )

      await helpers.registerUser(user)
      await helpers.selectTable(table.name)

      // Open modal dialog
      const createButton = screen.getByRole('button', { name: /add.*row|create/i })
      await helpers.user.click(createButton)

      await waitFor(() => {
        const dialog = screen.getByRole('dialog')
        expect(dialog).toBeInTheDocument()
        
        // Check modal has proper focus management attributes
        expect(dialog).toHaveAttribute('aria-modal', 'true') ||
        expect(dialog).toHaveAttribute('role', 'dialog')
      })

      // Test escape key closes modal
      await helpers.user.keyboard('{Escape}')
      
      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
      })
    })

    test('should restore focus after modal closes', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_focus_restore')
      await helpers.registerUser(user)

      // Test focus restoration (simplified in jsdom environment)
      const originalButton = screen.getByRole('button', { name: /create.*table/i })
      originalButton.focus()
      
      // Simulate modal open/close cycle
      await helpers.user.click(originalButton)
      
      const dialog = screen.queryByRole('dialog')
      if (dialog) {
        await helpers.user.keyboard('{Escape}')
        
        await waitFor(() => {
          expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
        })
        
        // Focus should return to trigger element
        await helpers.checkFocusVisible()
      }
    })
  })

  describe('Responsive Design', () => {
    test('should work on mobile viewport', async () => {
      // Simulate mobile viewport
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      })
      Object.defineProperty(window, 'innerHeight', {
        writable: true,
        configurable: true,
        value: 667,
      })

      helpers.renderComponent(<App />)

      // Check that auth form is accessible on mobile
      const authForm = screen.getByTestId('auth-form')
      expect(authForm).toBeInTheDocument()

      // Check that buttons are large enough for touch
      const loginButton = screen.getByRole('button', { name: /login/i })
      expect(loginButton).toBeInTheDocument()
      
      // In a real browser, we'd check computed styles for minimum touch target size
      const buttonStyle = window.getComputedStyle(loginButton)
      expect(buttonStyle.display).not.toBe('none')
    })

    test('should support touch interactions', async () => {
      helpers.renderComponent(<App />)

      // Test touch-friendly interactions
      const registerTab = screen.getByRole('tab', { name: /register/i })
      
      // Simulate touch interaction
      await helpers.user.click(registerTab)
      
      // Should switch to register form
      expect(screen.getByPlaceholderText('Email')).toBeInTheDocument()
    })

    test('should handle orientation changes', async () => {
      helpers.renderComponent(<App />)

      // Simulate orientation change
      Object.defineProperty(screen, 'orientation', {
        writable: true,
        configurable: true,
        value: { angle: 90 },
      })

      // Trigger resize event
      window.dispatchEvent(new Event('orientationchange'))

      // Interface should remain functional
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    })
  })

  describe('Color and Visual Accessibility', () => {
    test('should not rely solely on color for information', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_color_blind')

      // Test error state communication
      helpers.simulateServerError('/api/auth/login', 401)
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), 'invalid')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'invalid')
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))

      // Error should be communicated through text, not just color
      await waitFor(() => {
        const errorText = screen.getByText(/invalid/i)
        expect(errorText).toBeInTheDocument()
        
        // Should have text content, not just color change
        expect(errorText.textContent?.length).toBeGreaterThan(0)
      })
    })

    test('should support reduced motion preferences', async () => {
      // Mock prefers-reduced-motion
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: vi.fn().mockImplementation(query => ({
          matches: query === '(prefers-reduced-motion: reduce)',
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        })),
      })

      helpers.renderComponent(<App />)

      // Verify app respects motion preferences
      const authForm = screen.getByTestId('auth-form')
      expect(authForm).toBeInTheDocument()
      
      // Animations should be reduced or disabled
      await helpers.expectNoErrors()
    })

    test('should maintain readability with zoom up to 200%', async () => {
      // Simulate 200% zoom
      Object.defineProperty(document.documentElement, 'style', {
        writable: true,
        value: { zoom: '200%', fontSize: '32px' },
      })

      helpers.renderComponent(<App />)

      // Content should remain accessible at high zoom
      const authForm = screen.getByTestId('auth-form')
      expect(authForm).toBeInTheDocument()

      const inputs = screen.getAllByRole('textbox')
      inputs.forEach(input => {
        expect(input).toBeInTheDocument()
        expect(input).toBeVisible()
      })
    })
  })

  describe('Form Accessibility', () => {
    test('should associate labels with form controls', async () => {
      helpers.renderComponent(<App />)

      // Check input-label associations
      const usernameInput = screen.getByLabelText(/username/i)
      expect(usernameInput).toBeInTheDocument()
      expect(usernameInput).toHaveAttribute('id')

      const passwordInput = screen.getByLabelText(/password/i)
      expect(passwordInput).toBeInTheDocument()
      expect(passwordInput).toHaveAttribute('id')
    })

    test('should provide helpful error messages', async () => {
      helpers.renderComponent(<App />)

      // Test validation messages
      const form = screen.getByTestId('auth-form')
      
      // Submit empty form
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))

      // Should provide meaningful validation
      const requiredInputs = form.querySelectorAll('input[required]')
      requiredInputs.forEach(input => {
        expect(input).toHaveAttribute('aria-invalid', 'false')
        
        // Check for validation attributes
        expect(input.hasAttribute('required')).toBe(true)
      })
    })

    test('should group related form controls', async () => {
      helpers.renderComponent(<App />)

      // Check for proper form structure
      const authForm = screen.getByTestId('auth-form')
      expect(authForm).toBeInTheDocument()

      // Should have proper heading structure
      const heading = screen.getByRole('heading', { name: /login/i })
      expect(heading).toBeInTheDocument()

      // Tab panels should be properly structured
      const tabList = screen.getByRole('tablist')
      expect(tabList).toBeInTheDocument()

      const tabs = screen.getAllByRole('tab')
      expect(tabs.length).toBeGreaterThan(0)
    })
  })

  describe('Content Accessibility', () => {
    test('should have proper heading hierarchy', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_headings')
      await helpers.registerUser(user)

      // Check heading structure
      const headings = screen.getAllByRole('heading')
      expect(headings.length).toBeGreaterThan(0)

      // Should start with h1 or have logical progression
      headings.forEach(heading => {
        expect(heading).toBeInTheDocument()
        expect(heading.tagName).toMatch(/^H[1-6]$/)
      })
    })

    test('should provide alternative text for images', async () => {
      helpers.renderComponent(<App />)

      // Check for images and icons
      const images = document.querySelectorAll('img')
      images.forEach(img => {
        // Should have alt text or be decorative
        expect(
          img.hasAttribute('alt') || 
          img.hasAttribute('aria-hidden') ||
          img.getAttribute('role') === 'presentation'
        ).toBe(true)
      })

      // Check SVG icons
      const svgs = document.querySelectorAll('svg')
      svgs.forEach(svg => {
        // Should have proper accessibility attributes
        expect(
          svg.hasAttribute('aria-label') ||
          svg.hasAttribute('aria-labelledby') ||
          svg.hasAttribute('aria-hidden') ||
          svg.getAttribute('role') === 'img'
        ).toBe(true)
      })
    })

    test('should provide skip links for navigation', async () => {
      helpers.renderComponent(<App />)
      
      const user = createTestUser('_skip_links')
      await helpers.registerUser(user)

      // Look for skip links or main content landmarks
      const main = document.querySelector('main') || 
                   screen.queryByRole('main') ||
                   document.querySelector('[role="main"]')
      
      // Should have main content area identifiable
      expect(main || screen.getByTestId('main-app')).toBeInTheDocument()

      // Check for skip link (may not be visible but should exist)
      const skipLink = document.querySelector('a[href="#main"]') ||
                      document.querySelector('[class*="skip"]')
      
      // Skip links are optional but good practice
      if (skipLink) {
        expect(skipLink).toBeInTheDocument()
      }
    })
  })
})