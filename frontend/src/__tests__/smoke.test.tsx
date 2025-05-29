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
    
    expect(screen.getByTestId('auth-form')).toBeInTheDocument()
  })

  test('Authentication form is functional', async () => {
    helpers.renderComponent(<App />)
    
    expect(screen.getByRole('tab', { name: /login/i })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /register/i })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Username')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
  })

  test('Tab switching works correctly', async () => {
    helpers.renderComponent(<App />)
    
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
    
    await helpers.user.click(screen.getByRole('tab', { name: /register/i }))
    expect(screen.getByRole('button', { name: /register/i })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Email')).toBeInTheDocument()
    
    await helpers.user.click(screen.getByRole('tab', { name: /login/i }))
    expect(screen.getByRole('button', { name: /login/i })).toBeInTheDocument()
  })

  test('User registration flow works', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke')
    await helpers.registerUser(user)
    
    expect(screen.getByTestId('main-app')).toBeInTheDocument()
    expect(screen.getByText('Tables')).toBeInTheDocument()
  })

  test('User login flow works', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_login')
    
    await helpers.registerUser(user)
    await helpers.logout()
    await helpers.loginUser(user)
    
    expect(screen.getByTestId('main-app')).toBeInTheDocument()
  })

  test('Main app functionality is accessible after login', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_main')
    await helpers.registerUser(user)
    
    expect(screen.getByText('Tables')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create table/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /logout/i })).toBeInTheDocument()
  })

  test('Basic table creation works', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_table')
    await helpers.registerUser(user)
    
    const createTableButton = screen.getByRole('button', { name: /create table/i })
    await helpers.user.click(createTableButton)
    
    await waitFor(() => {
      const modal = screen.queryByRole('dialog')
      const newContent = screen.queryByText(/new table/i)
      expect(modal || newContent).toBeTruthy()
    })
  })

  test('API integration works without errors', async () => {
    helpers.renderComponent(<App />)
    
    const user = createTestUser('_smoke_api')
    await helpers.registerUser(user)
    
    await helpers.expectNoErrors()
    expect(screen.getByText('Tables')).toBeInTheDocument()
  })
})