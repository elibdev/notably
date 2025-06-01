import {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  HealthResponse,
  UserInfoResponse,
  TableResponse,
  TableListResponse,
  CreateTableRequest,
  UpdateTableRequest,
  EntityResponse,
  EntityListResponse,
  CreateEntityRequest,
  UpdateEntityRequest,
  FieldHistoryResponse,
  TableHistoryResponse,
  ErrorResponse,
  SuccessResponse,
  EntityQueryParams,
  HistoryQueryParams,
  ApiError,
} from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

class ApiService {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('auth_token');
  }

  setToken(token: string | null) {
    this.token = token;
    if (token) {
      localStorage.setItem('auth_token', token);
    } else {
      localStorage.removeItem('auth_token');
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    const config: RequestInit = {
      ...options,
      headers,
    };

    try {
      const response = await fetch(url, config);
      
      if (!response.ok) {
        const errorData: ErrorResponse = await response.json().catch(() => ({
          error: `HTTP ${response.status}: ${response.statusText}`,
        }));
        
        const apiError: ApiError = {
          message: errorData.error || `Request failed with status ${response.status}`,
          status: response.status,
        };
        throw apiError;
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }
      
      return {} as T;
    } catch (error) {
      if (error instanceof Error && 'status' in error) {
        throw error;
      }
      
      const apiError: ApiError = {
        message: error instanceof Error ? error.message : 'Network error',
        status: 0,
      };
      throw apiError;
    }
  }

  // Authentication endpoints
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
    this.setToken(response.token);
    return response;
  }

  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    });
    this.setToken(response.token);
    return response;
  }

  logout() {
    this.setToken(null);
  }

  // Health endpoints
  async getHealth(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health');
  }

  async getReady(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/ready');
  }

  // User endpoints
  async getUserInfo(): Promise<UserInfoResponse> {
    return this.request<UserInfoResponse>('/users/me');
  }

  async updateUserInfo(): Promise<SuccessResponse> {
    return this.request<SuccessResponse>('/users/me', {
      method: 'PUT',
    });
  }

  // Table endpoints
  async getTables(): Promise<TableListResponse> {
    return this.request<TableListResponse>('/tables');
  }

  async getTable(tableId: string): Promise<TableResponse> {
    return this.request<TableResponse>(`/tables/${tableId}`);
  }

  async createTable(data: CreateTableRequest): Promise<TableResponse> {
    return this.request<TableResponse>('/tables', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateTable(tableId: string, data: UpdateTableRequest): Promise<TableResponse> {
    return this.request<TableResponse>(`/tables/${tableId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteTable(tableId: string): Promise<SuccessResponse> {
    return this.request<SuccessResponse>(`/tables/${tableId}`, {
      method: 'DELETE',
    });
  }

  // Entity endpoints
  async getEntities(
    tableId: string,
    params: EntityQueryParams = {}
  ): Promise<EntityListResponse> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined) searchParams.append('limit', params.limit.toString());
    if (params.offset !== undefined) searchParams.append('offset', params.offset.toString());
    if (params.include_deleted !== undefined) searchParams.append('include_deleted', params.include_deleted.toString());
    
    const query = searchParams.toString();
    const endpoint = `/tables/${tableId}/entities${query ? `?${query}` : ''}`;
    
    return this.request<EntityListResponse>(endpoint);
  }

  async getEntity(
    tableId: string,
    entityId: string,
    includeDeleted?: boolean
  ): Promise<EntityResponse> {
    const params = includeDeleted ? '?include_deleted=true' : '';
    return this.request<EntityResponse>(`/tables/${tableId}/entities/${entityId}${params}`);
  }

  async createEntity(
    tableId: string,
    data: CreateEntityRequest
  ): Promise<EntityResponse> {
    return this.request<EntityResponse>(`/tables/${tableId}/entities`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateEntity(
    tableId: string,
    entityId: string,
    data: UpdateEntityRequest
  ): Promise<EntityResponse> {
    return this.request<EntityResponse>(`/tables/${tableId}/entities/${entityId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteEntity(tableId: string, entityId: string): Promise<SuccessResponse> {
    return this.request<SuccessResponse>(`/tables/${tableId}/entities/${entityId}`, {
      method: 'DELETE',
    });
  }

  async undeleteEntity(tableId: string, entityId: string): Promise<SuccessResponse> {
    return this.request<SuccessResponse>(`/tables/${tableId}/entities/${entityId}/undelete`, {
      method: 'POST',
    });
  }

  // History endpoints
  async getEntityHistory(
    tableId: string,
    entityId: string,
    params: HistoryQueryParams = {}
  ): Promise<TableHistoryResponse> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined) searchParams.append('limit', params.limit.toString());
    if (params.since !== undefined) searchParams.append('since', params.since);
    
    const query = searchParams.toString();
    const endpoint = `/tables/${tableId}/entities/${entityId}/history${query ? `?${query}` : ''}`;
    
    return this.request<TableHistoryResponse>(endpoint);
  }

  async getTableHistory(
    tableId: string,
    params: HistoryQueryParams = {}
  ): Promise<TableHistoryResponse> {
    const searchParams = new URLSearchParams();
    if (params.limit !== undefined) searchParams.append('limit', params.limit.toString());
    if (params.since !== undefined) searchParams.append('since', params.since);
    
    const query = searchParams.toString();
    const endpoint = `/tables/${tableId}/history${query ? `?${query}` : ''}`;
    
    return this.request<TableHistoryResponse>(endpoint);
  }

  async getFieldHistory(tableId: string, fieldName: string): Promise<FieldHistoryResponse> {
    return this.request<FieldHistoryResponse>(`/tables/${tableId}/history/fields/${fieldName}`);
  }

  // Helper methods
  isAuthenticated(): boolean {
    return !!this.token;
  }

  getToken(): string | null {
    return this.token;
  }
}

export const apiService = new ApiService();
export default apiService;