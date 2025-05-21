import { ApiClient, ColumnDefinition, TableInfo, RowData, RowEvent } from './api';

// Mock API responses
export interface MockResponses {
  listTables?: { tables: TableInfo[] };
  createTable?: TableInfo;
  listRows?: { rows: RowData[] };
  getRow?: RowData;
  createRow?: RowData;
  updateRow?: RowData;
  snapshot?: { rows: RowData[] };
  history?: { events: RowEvent[] };
}

// Factory functions to create test data
export const createMockTable = (
  name: string,
  columns: ColumnDefinition[] = []
): TableInfo => ({
  name,
  createdAt: new Date().toISOString(),
  columns,
});

export const createMockRow = (
  id: string,
  values: Record<string, unknown> = {}
): RowData => ({
  id,
  timestamp: new Date().toISOString(),
  values,
});

export const createMockEvent = (
  id: string,
  values: Record<string, unknown> | null = {}
): RowEvent => ({
  id,
  timestamp: new Date().toISOString(),
  values,
});

// Mock API client that doesn't make actual network requests
export class MockApiClient extends ApiClient {
  private mockResponses: MockResponses;
  
  constructor(apiKey: string = 'test-api-key', mockResponses: MockResponses = {}) {
    super(apiKey);
    this.mockResponses = mockResponses;
  }

  // Override API methods to return mock data
  async listTables(): Promise<{ tables: TableInfo[] }> {
    return (
      this.mockResponses.listTables || {
        tables: [createMockTable('MockTable', [
          { name: 'name', dataType: 'string' },
          { name: 'age', dataType: 'number' }
        ])]
      }
    );
  }

  async createTable(name: string, columns?: ColumnDefinition[]): Promise<TableInfo> {
    return this.mockResponses.createTable || createMockTable(name, columns);
  }

  async listRows(table: string): Promise<{ rows: RowData[] }> {
    return (
      this.mockResponses.listRows || {
        rows: [createMockRow('row1', { name: 'Test', age: 25 })]
      }
    );
  }
  
  async getRow(_table: string, id: string): Promise<RowData> {
    return this.mockResponses.getRow || createMockRow(id, { name: 'Test', age: 25 });
  }

  async createRow(_table: string, id?: string, values: Record<string, unknown> = {}): Promise<RowData> {
    return this.mockResponses.createRow || createMockRow(id || 'generated-id', values);
  }
  
  async updateRow(_table: string, id: string, values: Record<string, unknown>): Promise<RowData> {
    return this.mockResponses.updateRow || createMockRow(id, values);
  }

  async deleteRow(): Promise<void> {
    return Promise.resolve();
  }
  
  async snapshot(_table: string, _at?: string): Promise<{ rows: RowData[] }> {
    return (
      this.mockResponses.snapshot || {
        rows: [createMockRow('row1', { name: 'Test' })]
      }
    );
  }
  
  async history(_table: string, _start: string, _end: string): Promise<{ events: RowEvent[] }> {
    return (
      this.mockResponses.history || {
        events: [createMockEvent('row1', { name: 'Test' })]
      }
    );
  }
}

// Function to set up a test environment
export function setupApiTest(mockResponses: MockResponses = {}): MockApiClient {
  return new MockApiClient('test-api-key', mockResponses);
}

// Helper to check if columns are properly defined in tables
export function validateTableColumns(table: TableInfo): boolean {
  return Array.isArray(table.columns) && table.columns.length > 0;
}

// Debug helper that logs API interactions
export function createLoggingApiClient(client: ApiClient): ApiClient {
  const handler = {
    get(target: ApiClient, prop: string) {
      const original = target[prop as keyof ApiClient];
      if (typeof original === 'function') {
        return async (...args: unknown[]) => {
          console.log(`API Call: ${prop}`, args);
          try {
            const result = await original.apply(target, args);
            console.log(`API Result: ${prop}`, result);
            return result;
          } catch (error) {
            console.error(`API Error: ${prop}`, error);
            throw error;
          }
        };
      }
      return original;
    }
  };
  
  return new Proxy(client, handler);
}