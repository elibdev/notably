export interface ColumnDefinition {
  name: string;
  dataType: string;
}

// Error type for enhanced error objects
interface ErrorWithDetails extends Error {
  status?: number;
  responseData?: unknown;
}

export interface TableInfo {
  name: string;
  createdAt: string;
  columns?: ColumnDefinition[];
}

export interface RowData {
  id: string;
  timestamp: string;
  values: Record<string, unknown>;
}

export interface RowEvent {
  id: string;
  timestamp: string;
  values: Record<string, unknown> | null;
}

export interface RegisterResponse {
  id: string;
  username: string;
  email: string;
  apiKey: string;
}

// Use type alias instead of empty interface extension
export type LoginResponse = RegisterResponse;

async function handleResponse(response: Response) {
  const contentType = response.headers.get("Content-Type") || "";
  const isJson = contentType.includes("application/json");
  console.log(`Response status: ${response.status}, Content-Type: ${contentType}`);
  const text = await response.text();
  console.log(`Response body: ${text.substring(0, 200)}${text.length > 200 ? '...' : ''}`);
  const data = isJson ? JSON.parse(text) : { message: text };
  if (!response.ok) {
    const errorMsg = data.error || data.message || "Unknown API error";
    console.error(`API Error (${response.status}):`, errorMsg);
    // Preserve original error message but add status code for better debugging
    const error = new Error(`${errorMsg} (Status ${response.status})`);
    // Add additional properties to the error object for debugging
    (error as ErrorWithDetails).status = response.status;
    (error as ErrorWithDetails).responseData = data;
    throw error;
  }
  return data;
}

export class ApiClient {
  private apiKey: string;
  
  constructor(apiKey: string) {
    this.apiKey = apiKey;
    console.log(`ApiClient initialized with key: ${apiKey ? apiKey.substring(0, 5) + '...' : 'none'}`);
  }

  private headers() {
    const headers = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.apiKey}`,
    };
    console.log("Request headers:", { ...headers, Authorization: "Bearer [REDACTED]" });
    return headers;
  }

  static async register(
    username: string,
    email: string,
    password: string,
  ): Promise<RegisterResponse> {
    console.log(`Attempting to register user: ${username}, email: ${email}`);
    const res = await fetch("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, email, password }),
    });
    console.log(`Register response status: ${res.status}`);
    return handleResponse(res);
  }

  static async login(username: string, password: string): Promise<LoginResponse> {
    console.log("Attempting login for:", username);
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    console.log("Login response status:", res.status);
    console.log("Login response headers:", Object.fromEntries([...res.headers.entries()]));
    return handleResponse(res);
  }

  async listTables(): Promise<{ tables: TableInfo[] }> {
    console.log("Fetching tables list");
    const res = await fetch("/api/tables", {
      headers: this.headers(),
    });
    console.log(`Tables list response status: ${res.status}`);
    const data = await handleResponse(res);
    // Log column information for each table
    console.log("Tables with columns:", data.tables.map(t => ({
      name: t.name, 
      columnsCount: t.columns?.length || 0,
      columns: t.columns
    })));
    return data;
  }

  async createTable(name: string, columns?: ColumnDefinition[]): Promise<TableInfo> {
    console.log("Creating table with columns:", columns);
    const res = await fetch("/api/tables", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ name, columns }),
    });
    const data = await handleResponse(res);
    console.log("Table creation response:", data);
    return data;
  }

  async listRows(table: string): Promise<{ rows: RowData[] }> {
    const res = await fetch(`/api/tables/${encodeURIComponent(table)}/rows`, {
      headers: this.headers(),
    });
    return handleResponse(res);
  }

  async getRow(table: string, id: string): Promise<RowData> {
    const res = await fetch(
      `/api/tables/${encodeURIComponent(table)}/rows/${encodeURIComponent(id)}`,
      { headers: this.headers() },
    );
    return handleResponse(res);
  }

  async createRow(table: string, id?: string, values: Record<string, unknown>): Promise<RowData> {
    // If ID is undefined or empty, just send values and let backend generate the ID
    const payload = id && id.trim() ? { id, values } : { values };
    const res = await fetch(`/api/tables/${encodeURIComponent(table)}/rows`, {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify(payload),
    });
    return handleResponse(res);
  }

  async updateRow(table: string, id: string, values: Record<string, unknown>): Promise<RowData> {
    if (!id) {
      throw new Error("Row ID is required for updating a row");
    }
    const res = await fetch(
      `/api/tables/${encodeURIComponent(table)}/rows/${encodeURIComponent(id)}`,
      {
        method: "PUT",
        headers: this.headers(),
        body: JSON.stringify({ values }),
      },
    );
    return handleResponse(res);
  }

  async deleteRow(table: string, id: string): Promise<void> {
    if (!id) {
      throw new Error("Row ID is required for deleting a row");
    }
    const res = await fetch(
      `/api/tables/${encodeURIComponent(table)}/rows/${encodeURIComponent(id)}`,
      { method: "DELETE", headers: this.headers() },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error || "Failed to delete row");
    }
  }

  async snapshot(table: string, at?: string): Promise<{ rows: RowData[] }> {
    let path = `/api/tables/${encodeURIComponent(table)}/snapshot`;
    if (at) {
      path += `?at=${encodeURIComponent(at)}`;
    }
    const res = await fetch(path, { headers: this.headers() });
    return handleResponse(res);
  }

  async history(table: string, start: string, end: string): Promise<{ events: RowEvent[] }> {
    const params = new URLSearchParams({ start, end });
    const res = await fetch(
      `/api/tables/${encodeURIComponent(table)}/history?${params.toString()}`,
      { headers: this.headers() },
    );
    return handleResponse(res);
  }
}
