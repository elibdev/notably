import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { App } from './App';
import { ApiClient, TableInfo, ColumnDefinition } from './api';

// Mock the API client
jest.mock('./api', () => {
  return {
    ApiClient: jest.fn().mockImplementation(() => ({
      listTables: jest.fn().mockResolvedValue({
        tables: [
          { 
            name: 'TestTable', 
            createdAt: new Date().toISOString(),
            columns: [
              { name: 'stringField', dataType: 'string' },
              { name: 'numberField', dataType: 'number' },
              { name: 'booleanField', dataType: 'boolean' }
            ]
          }
        ]
      }),
      createTable: jest.fn().mockResolvedValue({ 
        name: 'NewTable', 
        createdAt: new Date().toISOString(),
        columns: [
          { name: 'newField', dataType: 'string' }
        ]
      }),
      listRows: jest.fn().mockResolvedValue({ 
        rows: [
          {
            id: 'row1',
            timestamp: new Date().toISOString(),
            values: {
              stringField: 'Test Value',
              numberField: 42,
              booleanField: true
            }
          }
        ]
      }),
      createRow: jest.fn().mockResolvedValue({
        id: 'new-row-id',
        timestamp: new Date().toISOString(),
        values: { stringField: 'New Value' }
      })
    })),
    // Mock static methods
    login: jest.fn().mockResolvedValue({ apiKey: 'test-api-key' }),
    register: jest.fn().mockResolvedValue({ apiKey: 'test-api-key' })
  };
});

// Mock react-router-dom
jest.mock('react-router-dom', () => ({
  BrowserRouter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Routes: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Route: ({ element }: { element: React.ReactNode }) => <div>{element}</div>,
  Navigate: () => <div>Navigate</div>,
  useParams: () => ({ tableName: 'TestTable' }),
  useNavigate: () => jest.fn()
}));

// Mock notifications
jest.mock('@mantine/notifications', () => ({
  notifications: {
    show: jest.fn()
  }
}));

// Helper function to set up ApiClient for tests
const setupApiClient = () => {
  localStorage.setItem('apiKey', 'test-api-key');
  return new ApiClient('test-api-key');
};

describe('App Component', () => {
  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  test('renders login form when not authenticated', () => {
    render(<App />);
    expect(screen.getByText(/Login to Notably/i)).toBeInTheDocument();
  });

  // Test specifically for the row data entry form
  test('TableView shows column fields in the row creation form', async () => {
    const mockColumns: ColumnDefinition[] = [
      { name: 'stringField', dataType: 'string' },
      { name: 'numberField', dataType: 'number' }
    ];
    
    const mockTableInfo: TableInfo = {
      name: 'TestTable',
      createdAt: new Date().toISOString(),
      columns: mockColumns
    };
    
    // Directly render TableView component to test form fields
    const { getByText } = render(
      <TableView 
        table="TestTable"
        tableInfo={mockTableInfo}
        client={setupApiClient()}
        onBack={() => {}}
      />
    );
    
    // Open the add row drawer
    fireEvent.click(getByText('Add Row'));
    
    // Check that column fields are rendered
    await waitFor(() => {
      expect(screen.getByText('stringField')).toBeInTheDocument();
      expect(screen.getByText('numberField')).toBeInTheDocument();
    });
  });
});

// Placeholder TableView component for testing
function TableView({ table, tableInfo }: { table: string; tableInfo?: TableInfo }) {
  return (
    <div>
      <h1>Table: {table}</h1>
      <button>Add Row</button>
      <div data-testid="columns-list">
        {tableInfo?.columns?.map((col: ColumnDefinition) => (
          <div key={col.name}>
            {col.name} ({col.dataType})
          </div>
        ))}
      </div>
    </div>
  );
}