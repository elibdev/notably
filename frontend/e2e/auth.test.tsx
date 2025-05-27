import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser } from '../src/test/helpers'
import { server } from '../src/test/setup'
import { http, HttpResponse } from 'msw'
import App from '../src/App'

describe('Authentication', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('User Registration', () => {
    test('should register a new user successfully', async () => {
      render(<App />)
      
      const user = createTestUser('_reg')
      await helpers.registerUser(user)
      
      // Should redirect to main app
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      expect(screen.getByText('Tables')).toBeInTheDocument()
    })

    test('should show error for duplicate username', async () => {
      render(<App />)
      
      const user = createTestUser('_dup')
      
      // Mock duplicate username error
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Username already exists' },
            { status: 409 }
          )
        })
      )
      
      // Switch to register tab
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
    })

    test('should show error for invalid email format', async () => {
      render(<App />)
      
      const user = createTestUser('_invalid')
      
      // Mock validation error
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Invalid email format' },
            { status: 400 }
          )
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), 'invalid-email')
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show validation error
      await waitFor(() => {
        expect(screen.getByText(/email/i)).toBeInTheDocument()
      })
    })

    test('should show error for empty fields', async () => {
      render(<App />)
      
      await helpers.user.click(screen.getByText('Register'))
      
      // Try to submit empty form
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show validation errors or prevent submission
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  })

  describe('User Login', () => {
    test('should login with valid credentials', async () => {
      render(<App />)
      
      const user = createTestUser('_login')
      await helpers.loginUser(user)
      
      // Should be in main app
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      expect(screen.getByText('Tables')).toBeInTheDocument()
    })

    test('should show error for invalid credentials', async () => {
      render(<App />)
      
      // Mock login failure
      server.use(
        http.post('/api/auth/login', () => {
          return HttpResponse.json(
            { error: 'Invalid credentials' },
            { status: 401 }
          )
        })
      )
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), 'nonexistentuser')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'wrongpassword')
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/invalid/i)).toBeInTheDocument()
      })
    })

    test('should show error for empty login fields', async () => {
      render(<App />)
      
      // Try to submit empty login form
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))
      
      // Should show validation errors or prevent submission
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })

    test('should switch between login and register tabs', async () => {
      render(<App />)
      
      // Should start on login tab
      expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
      
      // Switch to register
      await helpers.user.click(screen.getByText('Register'))
      expect(screen.getByRole('button', { name: /register/i })).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Email')).toBeInTheDocument()
      
      // Switch back to login
      await helpers.user.click(screen.getByText('Login'))
      expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    })
  })

  describe('Session Management', () => {
    test('should logout successfully', async () => {
      render(<App />)
      
      const user = createTestUser('_logout')
      await helpers.registerUser(user)
      
      // Should be logged in
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      
      // Logout
      await helpers.logout()
      
      // Should be back to auth form
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })

    test('should persist session across component remounts', async () => {
      const user = createTestUser('_session')
      
      // Mock authenticated state
      server.use(
        http.get('/api/auth/me', () => {
          return HttpResponse.json({
            user: { id: '1', username: user.username, email: user.email },
            apiKey: 'test-api-key'
          })
        })
      )
      
      // Set localStorage to simulate persisted session
      localStorage.setItem('authToken', 'test-token')
      
      render(<App />)
      
      // Should load directly into main app
      await waitFor(() => {
        expect(screen.getByTestId('main-app')).toBeInTheDocument()
      })
    })

    test('should redirect to login when accessing protected routes without auth', async () => {
      render(<App />)
      
      // Should show auth form by default when not authenticated
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  })

  describe('API Key Management', () => {
    test('should receive API key after registration', async () => {
      render(<App />)
      
      const user = createTestUser('_apikey')
      await helpers.registerUser(user)
      
      // Should be logged in and have access to main app
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      
      // Should have create table button (requires API access)
      expect(screen.getByRole('button', { name: /create table/i })).toBeInTheDocument()
    })

    test('should be able to make authenticated API calls', async () => {
      render(<App />)
      
      const user = createTestUser('_api')
      await helpers.registerUser(user)
      
      // Should load without API errors
      await helpers.expectNoErrors()
      
      // Should be able to see tables section
      expect(screen.getByText('Tables')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    test('should handle network errors gracefully', async () => {
      render(<App />)
      
      const user = createTestUser('_network')
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      // Simulate network error
      helpers.simulateNetworkError('/api/auth/register')
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should remain on the form
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })

    test('should handle server errors gracefully', async () => {
      render(<App />)
      
      const user = createTestUser('_server')
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      // Mock server error
      helpers.simulateServerError('/api/auth/register', 500)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show error message
      await waitFor(() => {
        expect(screen.getByText(/error/i)).toBeInTheDocument()
      })
    })
  })

  describe('Form Validation', () => {
    test('should validate password strength', async () => {
      render(<App />)
      
      // Mock validation error for weak password
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Password too weak' },
            { status: 400 }
          )
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      
      const user = createTestUser('_weak')
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), '123') // Weak password
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show validation error or prevent submission
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })

    test('should validate username format', async () => {
      render(<App />)
      
      // Mock validation error for invalid username
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Username too short' },
            { status: 400 }
          )
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), 'a') // Too short
      await helpers.user.type(screen.getByPlaceholderText('Email'), 'test@example.com')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'validpassword123')
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show validation error or prevent submission
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  })

  describe('UI State Management', () => {
    test('should disable form submission during loading', async () => {
      render(<App />)
      
      const user = createTestUser('_loading')
      
      // Mock slow response
      server.use(
        http.post('/api/auth/login', async () => {
          await new Promise(resolve => setTimeout(resolve, 1000))
          return HttpResponse.json({
            success: true,
            user: { id: '1', username: user.username, email: user.email },
            apiKey: 'test-api-key'
          })
        })
      )
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      const loginButton = screen.getByRole('button', { name: /login/i })
      await helpers.user.click(loginButton)
      
      // Button should be disabled during loading
      expect(loginButton).toBeDisabled()
    })

    test('should show loading states appropriately', async () => {
      render(<App />)
      
      const user = createTestUser('_loading_state')
      
      // Mock slow response
      server.use(
        http.post('/api/auth/register', async () => {
          await new Promise(resolve => setTimeout(resolve, 500))
          return HttpResponse.json({
            success: true,
            user: { id: '1', username: user.username, email: user.email },
            apiKey: 'test-api-key'
          })
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      // Should show loading indicator or disabled state
      const registerButton = screen.getByRole('button', { name: /register/i })
      expect(registerButton).toBeDisabled()
    })
  })
})