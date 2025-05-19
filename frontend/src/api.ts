export interface TableInfo {
  name: string;
  createdAt: string;
}

export interface RowData {
  id: string;
  timestamp: string;
  values: Record<string, any>;
}

export interface RowEvent {
  id: string;
  timestamp: string;
  values: Record<string, any> | null;
}

export interface RegisterResponse {
  id: string;
  username: string;
  email: string;
  apiKey: string;
}

export interface LoginResponse extends RegisterResponse {}

async function handleResponse(response: Response) {
  const contentType = response.headers.get("Content-Type") || "";
  const isJson = contentType.includes("application/json");
  const text = await response.text();
  const data = isJson ? JSON.parse(text) : { message: text };
  if (!response.ok) {
    throw new Error(data.error || data.message || "API error");
  }
  return data;
}

export class ApiClient {
  private apiKey: string;
  
  constructor(apiKey: string) {
    this.apiKey = apiKey;
  }

  private headers() {
    return {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.apiKey}`,
    };
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
    console.log("Attempting login for:", username);
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    console.log("Login response status:", res.status);
    return handleResponse(res);
  }

  async listTables(): Promise<{ tables: TableInfo[] }> {
    const res = await fetch("/api/tables", {
      headers: this.headers(),
    });
    return handleResponse(res);
  }

  async createTable(name: string): Promise<TableInfo> {
    const res = await fetch("/api/tables", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ name }),
    });
    return handleResponse(res);
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

  async createRow(table: string, id: string, values: Record<string, any>): Promise<RowData> {
    const res = await fetch(`/api/tables/${encodeURIComponent(table)}/rows`, {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ id, values }),
    });
    return handleResponse(res);
  }

  async updateRow(table: string, id: string, values: Record<string, any>): Promise<RowData> {
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
