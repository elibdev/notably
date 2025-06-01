// Base API types based on swagger definitions

export interface AuthResponse {
  expires_at: string;
  token: string;
  user_id: string;
}

export interface LoginRequest {
  password: string;
  user_id: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  user_id: string;
}

export interface ErrorResponse {
  error: string;
}

export interface SuccessResponse {
  message: string;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  version: string;
}

export interface UserInfoResponse {
  created_at: string;
  updated_at: string;
  user_id: string;
}

// Field types
export type FieldDataType = 
  | 'string'
  | 'number'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'text'
  | 'integer'
  | 'float';

export interface FieldRequest {
  data_type: FieldDataType;
  name: string;
}

export interface FieldResponse {
  data_type: string;
  name: string;
}

// Table types
export interface CreateTableRequest {
  fields: FieldRequest[];
  id: string;
}

export interface UpdateTableRequest {
  fields: FieldRequest[];
}

export interface TableResponse {
  created_at: string;
  fields: FieldResponse[];
  id: string;
  updated_at: string;
  user_id: string;
}

export interface TableListResponse {
  tables: TableResponse[];
}

// Entity types
export interface CreateEntityRequest {
  fields: Record<string, any>;
}

export interface UpdateEntityRequest {
  fields: Record<string, any>;
}

export interface EntityResponse {
  created_at: string;
  deleted_at?: string;
  entity_id: string;
  fields: Record<string, any>;
  is_deleted: boolean;
  table_id: string;
  timestamp: string;
}

export interface EntityListResponse {
  entities: EntityResponse[];
}

// History types
export interface FieldChangeResponse {
  entity_id: string;
  new_value: any;
  old_value: any;
  timestamp: string;
}

export interface FieldHistoryResponse {
  changes: FieldChangeResponse[];
  field_name: string;
  table_id: string;
}

export interface TableHistoryResponse {
  changes: FieldChangeResponse[];
  table_id: string;
}

// API Error types
export interface ApiError {
  message: string;
  status: number;
}

// Query parameters
export interface EntityQueryParams {
  limit?: number;
  offset?: number;
  include_deleted?: boolean;
}

export interface HistoryQueryParams {
  limit?: number;
  since?: string;
}

// Authentication context
export interface AuthContext {
  user: UserInfoResponse | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => void;
}

// API response wrapper
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  status: number;
}