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
  if (response.status === 204) { // Handle No Content
    return Promise.resolve(null);
  }

  const contentType = response.headers.get("Content-Type") || "";
  const isJson = contentType.includes("application/json");
  const text = await response.text();
  
  // Safely parse JSON: if text is empty, default to an empty object or an object with the text.
  let data;
  if (isJson) {
    try {
      data = text ? JSON.parse(text) : {}; // Handle empty text for JSON
    } catch (e) {
      // If JSON parsing fails for non-empty text, capture error and original text
      data = { message: `Failed to parse JSON response: ${text.substring(0,100)}`, parseError: e.message };
    }
  } else {
    data = { message: text };
  }

  if (!response.ok) {
    const errorMsg = data.error || data.message || "Unknown API error";
    const error = new Error(`${errorMsg} (Status ${response.status})`);
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
  }

  private headers() {
    const headers = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.apiKey}`,
    };
    return headers;
  }

  static async register(
    username: string,
    email: string,
    password: string,
  ): Promise<RegisterResponse> {
    const res = await fetch("/api/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, email, password }),
    });
    return handleResponse(res);
  }

  static async login(username: string, password: string): Promise<LoginResponse> {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    return handleResponse(res);
  }

  async listTables(): Promise<{ tables: TableInfo[] }> {
    const res = await fetch("/api/tables", {
      headers: this.headers(),
    });
    const data = await handleResponse(res);
    return data;
  }

  async createTable(name: string, columns?: ColumnDefinition[]): Promise<TableInfo> {
    const res = await fetch("/api/tables", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ name, columns }),
    });
    const data = await handleResponse(res);
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
    // Use handleResponse for consistent error handling.
    // The return value of handleResponse is ignored here as deleteRow is Promise<void>.
    await handleResponse(res);
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
