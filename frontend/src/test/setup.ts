import '@testing-library/jest-dom'
import { expect, afterEach, beforeAll, afterAll } from 'vitest'
import { cleanup } from '@testing-library/react'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

// Setup MSW server for API mocking
export const server = setupServer(
  // Default handlers - can be overridden in individual tests
  http.post('/api/auth/register', () => {
    return HttpResponse.json({
      success: true,
      user: { id: '1', username: 'testuser', email: 'test@example.com' },
      apiKey: 'test-api-key'
    })
  }),
  
  http.post('/api/auth/login', () => {
    return HttpResponse.json({
      success: true,
      user: { id: '1', username: 'testuser', email: 'test@example.com' },
      apiKey: 'test-api-key'
    })
  }),
  
  http.post('/api/auth/logout', () => {
    return HttpResponse.json({ success: true })
  }),
  
  http.get('/api/tables', () => {
    return HttpResponse.json([
      { id: '1', name: 'Test Table', createdAt: '2024-01-01T00:00:00Z' }
    ])
  }),
  
  http.post('/api/tables', () => {
    return HttpResponse.json({
      id: '2',
      name: 'New Table',
      createdAt: new Date().toISOString()
    }, { status: 201 })
  }),
  
  http.get('/api/tables/:id/rows', () => {
    return HttpResponse.json([
      { id: '1', values: { name: 'Test Row', value: 'Test Value' } }
    ])
  }),
  
  http.post('/api/tables/:id/rows', () => {
    return HttpResponse.json({
      id: '2',
      values: { name: 'New Row', value: 'New Value' }
    }, { status: 201 })
  })
)

// Establish API mocking before all tests
beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
})

// Clean up after each test
afterEach(() => {
  cleanup()
  server.resetHandlers()
})

// Clean up after all tests
afterAll(() => {
  server.close()
})

// Mock window.matchMedia for Mantine components
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // deprecated
    removeListener: vi.fn(), // deprecated
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// Mock ResizeObserver
global.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// Mock IntersectionObserver
global.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

// Setup localStorage mock
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
global.localStorage = localStorageMock

// Setup sessionStorage mock
const sessionStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
global.sessionStorage = sessionStorageMock

// Mock fetch if not already available
if (!global.fetch) {
  global.fetch = vi.fn()
}

// Import jest-dom matchers
import * as matchers from '@testing-library/jest-dom/matchers'

// Extend expect with jest-dom matchers
expect.extend(matchers)