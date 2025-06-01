import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock API client class that would typically be in a separate file
class ApiClient {
  constructor(baseUrl = '') {
    this.baseUrl = baseUrl
    this.token = null
  }

  setAuthToken(token) {
    this.token = token
  }

  async request(method, endpoint, data = null) {
    const url = `${this.baseUrl}${endpoint}`
    
    const headers = {
      'Content-Type': 'application/json'
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const config = {
      method,
      headers
    }

    if (data && method !== 'GET') {
      config.body = JSON.stringify(data)
    }

    const response = await fetch(url, config)
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }

    return await response.json()
  }

  // Auth endpoints
  async login(userID, password) {
    return this.request('POST', '/api/v1/auth/login', {
      user_id: userID,
      password
    })
  }

  async register(userID, email, password) {
    return this.request('POST', '/api/v1/auth/register', {
      user_id: userID,
      email,
      password
    })
  }

  // Table endpoints
  async getTables() {
    return this.request('GET', '/api/v1/tables')
  }

  async createTable(tableData) {
    return this.request('POST', '/api/v1/tables', tableData)
  }

  async getTable(tableId) {
    return this.request('GET', `/api/v1/tables/${tableId}`)
  }

  // Entity endpoints
  async getEntities(tableId) {
    return this.request('GET', `/api/v1/tables/${tableId}/entities`)
  }

  async createEntity(tableId, entityData) {
    return this.request('POST', `/api/v1/tables/${tableId}/entities`, entityData)
  }

  async updateEntity(tableId, entityId, entityData) {
    return this.request('PUT', `/api/v1/tables/${tableId}/entities/${entityId}`, entityData)
  }

  async deleteEntity(tableId, entityId) {
    return this.request('DELETE', `/api/v1/tables/${tableId}/entities/${entityId}`)
  }

  // Health check
  async health() {
    return this.request('GET', '/api/v1/health')
  }
}

describe('ApiClient', () => {
  let apiClient
  let mockFetch

  beforeEach(() => {
    apiClient = new ApiClient('http://localhost:8080')
    mockFetch = vi.fn()
    global.fetch = mockFetch
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('Basic Request Functionality', () => {
    it('makes GET requests correctly', async () => {
      const mockResponse = { status: 'healthy' }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.health()

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/health', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      expect(result).toEqual(mockResponse)
    })

    it('makes POST requests with data correctly', async () => {
      const requestData = { user_id: 'test', password: 'pass123' }
      const mockResponse = { token: 'abc123', user_id: 'test' }
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.login('test', 'pass123')

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestData)
      })
      expect(result).toEqual(mockResponse)
    })

    it('includes authorization header when token is set', async () => {
      apiClient.setAuthToken('test-token')
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ tables: [] })
      })

      await apiClient.getTables()

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:8080/api/v1/tables', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token'
        }
      })
    })
  })

  describe('Authentication Methods', () => {
    it('performs login correctly', async () => {
      const mockResponse = { token: 'login-token', user_id: 'testuser' }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.login('testuser', 'password123')

      expect(result).toEqual(mockResponse)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/login',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            user_id: 'testuser',
            password: 'password123'
          })
        })
      )
    })

    it('performs registration correctly', async () => {
      const mockResponse = { token: 'reg-token', user_id: 'newuser' }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.register('newuser', 'new@example.com', 'newpass123')

      expect(result).toEqual(mockResponse)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/register',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            user_id: 'newuser',
            email: 'new@example.com',
            password: 'newpass123'
          })
        })
      )
    })
  })

  describe('Table Operations', () => {
    beforeEach(() => {
      apiClient.setAuthToken('valid-token')
    })

    it('fetches tables list', async () => {
      const mockTables = { tables: [{ id: 'contacts', fields: [] }] }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockTables)
      })

      const result = await apiClient.getTables()

      expect(result).toEqual(mockTables)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables',
        expect.objectContaining({ method: 'GET' })
      )
    })

    it('creates new table', async () => {
      const tableData = {
        id: 'contacts',
        fields: [
          { name: 'name', data_type: 'string' },
          { name: 'email', data_type: 'string' }
        ]
      }
      const mockResponse = { id: 'contacts', created_at: '2024-01-01T00:00:00Z' }
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.createTable(tableData)

      expect(result).toEqual(mockResponse)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(tableData)
        })
      )
    })

    it('fetches specific table', async () => {
      const mockTable = { id: 'contacts', fields: [], entities: [] }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockTable)
      })

      const result = await apiClient.getTable('contacts')

      expect(result).toEqual(mockTable)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables/contacts',
        expect.objectContaining({ method: 'GET' })
      )
    })
  })

  describe('Entity Operations', () => {
    beforeEach(() => {
      apiClient.setAuthToken('valid-token')
    })

    it('fetches entities for a table', async () => {
      const mockEntities = { entities: [{ id: '123', name: 'John' }] }
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockEntities)
      })

      const result = await apiClient.getEntities('contacts')

      expect(result).toEqual(mockEntities)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables/contacts/entities',
        expect.objectContaining({ method: 'GET' })
      )
    })

    it('creates new entity', async () => {
      const entityData = { name: 'John Doe', email: 'john@example.com' }
      const mockResponse = { id: '123', ...entityData }
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.createEntity('contacts', entityData)

      expect(result).toEqual(mockResponse)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables/contacts/entities',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(entityData)
        })
      )
    })

    it('updates existing entity', async () => {
      const entityData = { name: 'Jane Doe', email: 'jane@example.com' }
      const mockResponse = { id: '123', ...entityData }
      
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockResponse)
      })

      const result = await apiClient.updateEntity('contacts', '123', entityData)

      expect(result).toEqual(mockResponse)
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables/contacts/entities/123',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(entityData)
        })
      )
    })

    it('deletes entity', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({})
      })

      await apiClient.deleteEntity('contacts', '123')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/tables/contacts/entities/123',
        expect.objectContaining({ method: 'DELETE' })
      )
    })
  })

  describe('Error Handling', () => {
    it('throws error for non-ok HTTP responses', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Not found' })
      })

      await expect(apiClient.health()).rejects.toThrow('Not found')
    })

    it('throws generic error when response has no error message', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({})
      })

      await expect(apiClient.health()).rejects.toThrow('HTTP 500')
    })

    it('handles malformed JSON error responses', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.reject(new Error('Invalid JSON'))
      })

      await expect(apiClient.health()).rejects.toThrow('HTTP 400')
    })

    it('handles network errors', async () => {
      mockFetch.mockRejectedValue(new Error('Network error'))

      await expect(apiClient.health()).rejects.toThrow('Network error')
    })
  })

  describe('Token Management', () => {
    it('starts without auth token', () => {
      expect(apiClient.token).toBeNull()
    })

    it('sets auth token correctly', () => {
      apiClient.setAuthToken('new-token')
      expect(apiClient.token).toBe('new-token')
    })

    it('can clear auth token', () => {
      apiClient.setAuthToken('token')
      apiClient.setAuthToken(null)
      expect(apiClient.token).toBeNull()
    })
  })

  describe('Request Configuration', () => {
    it('handles requests without data correctly', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({})
      })

      await apiClient.request('GET', '/test')

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:8080/test', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json'
        }
      })
    })

    it('excludes body for GET requests even when data is provided', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({})
      })

      await apiClient.request('GET', '/test', { some: 'data' })

      const call = mockFetch.mock.calls[0]
      expect(call[1]).not.toHaveProperty('body')
    })

    it('uses correct base URL', async () => {
      const customClient = new ApiClient('https://api.example.com')
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({})
      })

      await customClient.health()

      expect(mockFetch).toHaveBeenCalledWith(
        'https://api.example.com/api/v1/health',
        expect.any(Object)
      )
    })
  })
})