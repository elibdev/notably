import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser } from '../test/helpers'
import { server } from '../test/setup'
import { http, HttpResponse } from 'msw'
import App from '../App'

describe('Authentication', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  describe('Error Handling', () => {
    test('should show error for duplicate username', async () => {
      render(<App />)
      
      const user = createTestUser('_dup')
      
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Username already exists' },
            { status: 409 }
          )
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
    })

    test('should show error for invalid credentials', async () => {
      render(<App />)
      
      server.use(
        http.post('/api/auth/login', () => {
          return HttpResponse.json(
            { error: 'Invalid credentials' },
            { status: 401 }
          )
        })
      )
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), 'invaliduser')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'wrongpassword')
      await helpers.user.click(screen.getByRole('button', { name: /login/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/invalid/i)).toBeInTheDocument()
      })
    })

    test('should handle server errors gracefully', async () => {
      render(<App />)
      
      const user = createTestUser('_server')
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), user.username)
      await helpers.user.type(screen.getByPlaceholderText('Email'), user.email)
      await helpers.user.type(screen.getByPlaceholderText('Password'), user.password)
      
      helpers.simulateServerError('/api/auth/register', 500)
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/error/i)).toBeInTheDocument()
      })
    })
  })

  describe('Session Management', () => {
    test('should persist session across component remounts', async () => {
      const user = createTestUser('_session')
      
      server.use(
        http.get('/api/auth/me', () => {
          return HttpResponse.json({
            user: { id: '1', username: user.username, email: user.email },
            apiKey: 'test-api-key'
          })
        })
      )
      
      localStorage.setItem('authToken', 'test-token')
      
      render(<App />)
      
      await waitFor(() => {
        expect(screen.getByTestId('main-app')).toBeInTheDocument()
      })
    })

    test('should logout successfully', async () => {
      render(<App />)
      
      const user = createTestUser('_logout')
      await helpers.registerUser(user)
      
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      
      await helpers.logout()
      
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  })

  describe('Form Validation', () => {
    test('should validate email format', async () => {
      render(<App />)
      
      server.use(
        http.post('/api/auth/register', () => {
          return HttpResponse.json(
            { error: 'Invalid email format' },
            { status: 400 }
          )
        })
      )
      
      await helpers.user.click(screen.getByText('Register'))
      
      await helpers.user.type(screen.getByPlaceholderText('Username'), 'testuser')
      await helpers.user.type(screen.getByPlaceholderText('Email'), 'invalid-email')
      await helpers.user.type(screen.getByPlaceholderText('Password'), 'password123')
      
      await helpers.user.click(screen.getByRole('button', { name: /register/i }))
      
      await waitFor(() => {
        expect(screen.getByText(/email/i)).toBeInTheDocument()
      })
    })

    test('should disable form submission during loading', async () => {
      render(<App />)
      
      const user = createTestUser('_loading')
      
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
      
      expect(loginButton).toBeDisabled()
    })
  })

  describe('API Key Management', () => {
    test('should receive API key after registration', async () => {
      render(<App />)
      
      const user = createTestUser('_apikey')
      await helpers.registerUser(user)
      
      expect(screen.getByTestId('main-app')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /create table/i })).toBeInTheDocument()
    })
  })
})