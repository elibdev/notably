import React from 'react'
import { describe, test, expect, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { TestHelpers, createTestUser } from '../test/helpers'
import App from '../App'

describe('Smoke Tests', () => {
  let helpers: TestHelpers

  beforeEach(() => {
    helpers = new TestHelpers()
  })

  test('App renders without crashing', () => {
    helpers.renderComponent(<App />)
    
    // Should render auth form by default
    expect(screen.getByTestId('auth-form')).toBeInTheDocument()
  })

  test('Authentication form is functional', async () => {
    helpers.renderComponent(<App />)
    
    // Should have login and register tabs
    expect(screen.getByRole('tab', { name: /login/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /register/i })).toBeInTheDocument()
    
    // Should have input fields
    expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    
    // Should have submit button
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
  })

  test('Tab switching works correctly', async () => {
    helpers.renderComponent(<App />)
    
    // Start on login tab
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    
    // Switch to register tab
    await helpers.user.click(screen.getByRole('tab', { name: /register/i }))
    expect(screen.getByRole('button', { name: /register/i })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument()
    
    // Switch back to login
    await helpers.user.click(screen.getByRole('tab', { name: /login/i }))
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
  })

  test('User registration flow works', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke')
    await helpers.registerUser(user)
    
    // Should redirect to main app after successful registration
    expect(screen.getByTestId('main-app')).toBeInTheDocument()
    expect(screen.getByText('Tables')).toBeInTheDocument()
  })

  test('User login flow works', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_login')
    
    // First register user
    await helpers.registerUser(user)
    
    // Then logout
    await helpers.logout()
    
    // Then login again
    await helpers.loginUser(user)
    
    // Should be in main app
    expect(screen.getByTestId('main-app')).toBeInTheDocument()
  })

  test('Error handling works for invalid login', async () => {
    helpers.renderComponent(<App />)
    
    // Mock login failure
    helpers.simulateServerError('/api/auth/login', 401)
    
    await helpers.user.type(screen.getByPlaceholderText('Username'), 'invaliduser')
    await helpers.user.type(screen.getByPlaceholderText('Password'), 'wrongpassword')
    await helpers.user.click(screen.getByRole('button', { name: /login/i }))
    
    // Should show error message
    await waitFor(() => {
      expect(screen.getByText(/invalid/i)).toBeInTheDocument()
    })
  })

  test('Network error handling works', async () => {
    helpers.renderComponent(<App />)
    
    // Mock network error
    helpers.simulateNetworkError('/api/auth/login')
    
    await helpers.user.type(screen.getByPlaceholderText('Username'), 'testuser')
    await helpers.user.type(screen.getByPlaceholderText('Password'), 'password')
    await helpers.user.click(screen.getByRole('button', { name: /login/i }))
    
    // Should remain on auth form (login failed)
    await waitFor(() => {
      expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    })
  })

  test('Form validation prevents empty submissions', async () => {
    helpers.renderComponent(<App />)
    
    // Try to submit empty login form
    await helpers.user.click(screen.getByRole('button', { name: /login/i }))
    
    // Should remain on auth form
    expect(screen.getByTestId('auth-form')).toBeInTheDocument()
    
    // Switch to register and try empty submission
    await helpers.user.click(screen.getByRole('tab', { name: /register/i }))
    await helpers.user.click(screen.getByRole('button', { name: /register/i }))
    
    // Should remain on auth form
    expect(screen.getByTestId('auth-form')).toBeInTheDocument()
  })

  test('Main app functionality is accessible after login', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_main')
    await helpers.registerUser(user)
    
    // Should have main navigation
    expect(screen.getByText('Tables')).toBeInTheDocument()
    
    // Should have create table button
    expect(screen.getByRole('button', { name: /create table/i })).toBeInTheDocument()
    
    // Should have logout functionality
    expect(screen.getByRole('button', { name: /logout/i })).toBeInTheDocument()
  })

  test('API integration is working', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_api')
    await helpers.registerUser(user)
    
    // Should load without API errors
    await helpers.expectNoErrors()
    
    // Should be able to interact with tables
    expect(screen.getByText('Tables')).toBeInTheDocument()
  })

  test('Component interactions work correctly', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_interact')
    await helpers.registerUser(user)
    
    // Test button clicks work
    const createTableButton = screen.getByRole('button', { name: /create table/i })
    expect(createTableButton).toBeInTheDocument()
    
    // Test that we can focus elements
    await helpers.user.click(createTableButton)
    
    // Should open modal or navigate
    await waitFor(() => {
      // Either a modal opens or we navigate somewhere
      const modal = screen.queryByRole('dialog')
      const newContent = screen.queryByText(/new table/i)
      expect(modal || newContent).toBeTruthy()
    })
  })
})