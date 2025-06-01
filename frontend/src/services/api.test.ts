import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import apiService from './api';
import { 
  LoginRequest, 
  RegisterRequest, 
  CreateTableRequest, 
  CreateEntityRequest,
  ApiError
} from '../types/api';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock fetch
const mockFetch = vi.fn();
globalThis.fetch = mockFetch;

describe('ApiService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset API service token
    apiService.setToken(null);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Authentication', () => {
    it('should login successfully and store token', async () => {
      const loginRequest: LoginRequest = {
        user_id: 'testuser',
        password: 'password123',
      };

      const mockResponse = {
        token: 'mock-jwt-token',
        user_id: 'testuser',
        expires_at: '2024-12-31T23:59:59Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.login(loginRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/auth/login',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(loginRequest),
        })
      );

      expect(result).toEqual(mockResponse);
      expect(localStorageMock.setItem).toHaveBeenCalledWith('auth_token', 'mock-jwt-token');
      expect(apiService.getToken()).toBe('mock-jwt-token');
    });

    it('should register successfully and store token', async () => {
      const registerRequest: RegisterRequest = {
        user_id: 'newuser',
        email: 'test@example.com',
        password: 'password123',
      };

      const mockResponse = {
        token: 'mock-jwt-token',
        user_id: 'newuser',
        expires_at: '2024-12-31T23:59:59Z',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.register(registerRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/auth/register',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(registerRequest),
        })
      );

      expect(result).toEqual(mockResponse);
      expect(localStorageMock.setItem).toHaveBeenCalledWith('auth_token', 'mock-jwt-token');
    });

    it('should logout and clear token', () => {
      apiService.setToken('some-token');
      
      apiService.logout();
      
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('auth_token');
      expect(apiService.getToken()).toBeNull();
    });

    it('should handle login error', async () => {
      const loginRequest: LoginRequest = {
        user_id: 'testuser',
        password: 'wrongpassword',
      };

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: vi.fn().mockResolvedValue({ error: 'Invalid credentials' }),
      });

      await expect(apiService.login(loginRequest)).rejects.toEqual({
        message: 'Invalid credentials',
        status: 401,
      });
    });
  });

  describe('Token Management', () => {
    it('should initialize token from localStorage', () => {
      localStorageMock.getItem.mockReturnValue('stored-token');
      
      // Create new instance to test initialization
      const newApiService = new (apiService.constructor as any)();
      
      expect(localStorageMock.getItem).toHaveBeenCalledWith('auth_token');
      expect(newApiService.getToken()).toBe('stored-token');
    });

    it('should include Authorization header when token is set', async () => {
      apiService.setToken('test-token');

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ status: 'ok' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      await apiService.getHealth();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/health',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );
    });

    it('should not include Authorization header when no token', async () => {
      apiService.setToken(null);

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ status: 'ok' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      await apiService.getHealth();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/health',
        expect.objectContaining({
          headers: expect.not.objectContaining({
            'Authorization': expect.anything(),
          }),
        })
      );
    });
  });

  describe('Table Operations', () => {
    beforeEach(() => {
      apiService.setToken('test-token');
    });

    it('should fetch tables successfully', async () => {
      const mockTables = {
        tables: [
          {
            id: 'table1',
            fields: [{ name: 'name', data_type: 'string' }],
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z',
            user_id: 'testuser',
          },
        ],
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockTables),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.getTables();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/tables',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockTables);
    });

    it('should create table successfully', async () => {
      const createRequest: CreateTableRequest = {
        id: 'new-table',
        fields: [
          { name: 'name', data_type: 'string' },
          { name: 'age', data_type: 'number' },
        ],
      };

      const mockResponse = {
        id: 'new-table',
        fields: createRequest.fields,
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
        user_id: 'testuser',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.createTable(createRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/tables',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(createRequest),
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockResponse);
    });

    it('should delete table successfully', async () => {
      const tableId = 'table-to-delete';

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ message: 'Table deleted successfully' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.deleteTable(tableId);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}`,
        expect.objectContaining({
          method: 'DELETE',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual({ message: 'Table deleted successfully' });
    });
  });

  describe('Entity Operations', () => {
    beforeEach(() => {
      apiService.setToken('test-token');
    });

    it('should fetch entities with query parameters', async () => {
      const tableId = 'test-table';
      const mockEntities = {
        entities: [
          {
            entity_id: 'entity1',
            table_id: tableId,
            fields: { name: 'John', age: 30 },
            created_at: '2023-01-01T00:00:00Z',
            timestamp: '2023-01-01T00:00:00Z',
            is_deleted: false,
          },
        ],
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockEntities),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const params = { limit: 10, offset: 0, include_deleted: true };
      const result = await apiService.getEntities(tableId, params);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}/entities?limit=10&offset=0&include_deleted=true`,
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockEntities);
    });

    it('should create entity successfully', async () => {
      const tableId = 'test-table';
      const createRequest: CreateEntityRequest = {
        fields: { name: 'Jane', age: 25 },
      };

      const mockResponse = {
        entity_id: 'new-entity-id',
        table_id: tableId,
        fields: createRequest.fields,
        created_at: '2023-01-01T00:00:00Z',
        timestamp: '2023-01-01T00:00:00Z',
        is_deleted: false,
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.createEntity(tableId, createRequest);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}/entities`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(createRequest),
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockResponse);
    });

    it('should undelete entity successfully', async () => {
      const tableId = 'test-table';
      const entityId = 'entity-to-restore';

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ message: 'Entity restored successfully' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.undeleteEntity(tableId, entityId);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}/entities/${entityId}/undelete`,
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual({ message: 'Entity restored successfully' });
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(apiService.getHealth()).rejects.toEqual({
        message: 'Network error',
        status: 0,
      });
    });

    it('should handle HTTP errors with JSON response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: vi.fn().mockResolvedValue({ error: 'Invalid request data' }),
      });

      await expect(apiService.getHealth()).rejects.toEqual({
        message: 'Invalid request data',
        status: 400,
      });
    });

    it('should handle HTTP errors without JSON response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: vi.fn().mockRejectedValue(new Error('Not JSON')),
      });

      await expect(apiService.getHealth()).rejects.toEqual({
        message: 'HTTP 500: Internal Server Error',
        status: 500,
      });
    });

    it('should handle responses without content-type header', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers(),
        json: () => Promise.resolve({}),
      });

      const result = await apiService.getHealth();
      expect(result).toEqual({});
    });
  });

  describe('Helper Methods', () => {
    it('should correctly report authentication status', () => {
      expect(apiService.isAuthenticated()).toBe(false);
      
      apiService.setToken('test-token');
      expect(apiService.isAuthenticated()).toBe(true);
      
      apiService.setToken(null);
      expect(apiService.isAuthenticated()).toBe(false);
    });

    it('should return current token', () => {
      expect(apiService.getToken()).toBeNull();
      
      apiService.setToken('test-token');
      expect(apiService.getToken()).toBe('test-token');
    });
  });

  describe('History Operations', () => {
    beforeEach(() => {
      apiService.setToken('test-token');
    });

    it('should fetch entity history with parameters', async () => {
      const tableId = 'test-table';
      const entityId = 'test-entity';
      const mockHistory = {
        table_id: tableId,
        changes: [
          {
            entity_id: entityId,
            old_value: 'old',
            new_value: 'new',
            timestamp: '2023-01-01T00:00:00Z',
          },
        ],
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockHistory),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const params = { limit: 10, since: '2023-01-01T00:00:00Z' };
      const result = await apiService.getEntityHistory(tableId, entityId, params);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}/entities/${entityId}/history?limit=10&since=2023-01-01T00%3A00%3A00Z`,
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockHistory);
    });

    it('should fetch field history', async () => {
      const tableId = 'test-table';
      const fieldName = 'test-field';
      const mockHistory = {
        table_id: tableId,
        field_name: fieldName,
        changes: [],
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockHistory),
        headers: new Headers({ 'content-type': 'application/json' }),
      });

      const result = await apiService.getFieldHistory(tableId, fieldName);

      expect(mockFetch).toHaveBeenCalledWith(
        `http://localhost:8080/tables/${tableId}/history/fields/${fieldName}`,
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );

      expect(result).toEqual(mockHistory);
    });
  });
});