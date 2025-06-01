import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MantineProvider } from '@mantine/core'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock components for integration testing
// In a real app, these would be imported from their respective files

const useAuthStore = vi.fn()
const mockApiClient = {
  request: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
  getTables: vi.fn(),
  createTable: vi.fn(),
  getEntities: vi.fn(),
  createEntity: vi.fn()
}

// Simplified App component for testing
function TestApp() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [user, setUser] = useState(null)
  const [currentView, setCurrentView] = useState('auth')
  const [tables, setTables] = useState([])
  const [selectedTable, setSelectedTable] = useState(null)
  const [entities, setEntities] = useState([])

  const handleLogin = async (credentials) => {
    try {
      const response = await mockApiClient.login(credentials.user_id, credentials.password)
      setUser({ user_id: credentials.user_id, token: response.token })
      setIsAuthenticated(true)
      setCurrentView('dashboard')
      
      // Load tables after login
      const tablesResponse = await mockApiClient.getTables()
      setTables(tablesResponse.tables || [])
    } catch (error) {
      throw error
    }
  }

  const handleCreateTable = async (tableData) => {
    try {
      const newTable = await mockApiClient.createTable(tableData)
      setTables(prev => [...prev, newTable])
    } catch (error) {
      console.error('Failed to create table:', error)
      // In a real app, you'd show an error notification
    }
  }

  const handleSelectTable = async (tableId) => {
    setSelectedTable(tableId)
    setCurrentView('table')
    
    const entitiesResponse = await mockApiClient.getEntities(tableId)
    setEntities(entitiesResponse.entities || [])
  }

  const handleCreateEntity = async (entityData) => {
    if (!selectedTable) return
    
    try {
      const newEntity = await mockApiClient.createEntity(selectedTable, entityData)
      setEntities(prev => [...prev, newEntity])
    } catch (error) {
      console.error('Failed to create entity:', error)
      // In a real app, you'd show an error notification
    }
  }

  if (!isAuthenticated) {
    return (
      <div data-testid="auth-view">
        <AuthForm onSuccess={handleLogin} />
      </div>
    )
  }

  if (currentView === 'dashboard') {
    return (
      <div data-testid="dashboard-view">
        <div data-testid="user-info">Welcome, {user.user_id}</div>
        <button 
          data-testid="logout-button"
          onClick={() => {
            setIsAuthenticated(false)
            setUser(null)
            setCurrentView('auth')
          }}
        >
          Logout
        </button>
        
        <TablesDashboard 
          tables={tables}
          onSelectTable={handleSelectTable}
          onCreateTable={handleCreateTable}
        />
      </div>
    )
  }

  if (currentView === 'table') {
    return (
      <div data-testid="table-view">
        <button 
          data-testid="back-button"
          onClick={() => setCurrentView('dashboard')}
        >
          Back to Dashboard
        </button>
        
        <div data-testid="table-header">Table: {selectedTable}</div>
        
        <TableView 
          tableId={selectedTable}
          entities={entities}
          onCreateEntity={handleCreateEntity}
        />
      </div>
    )
  }

  return null
}

// Simplified component implementations
function AuthForm({ onSuccess }) {
  const [formData, setFormData] = useState({ user_id: '', password: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      await onSuccess(formData)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} data-testid="login-form">
      <input
        data-testid="user-id-input"
        type="text"
        placeholder="User ID"
        value={formData.user_id}
        onChange={(e) => setFormData(prev => ({ ...prev, user_id: e.target.value }))}
        required
      />
      <input
        data-testid="password-input"
        type="password"
        placeholder="Password"
        value={formData.password}
        onChange={(e) => setFormData(prev => ({ ...prev, password: e.target.value }))}
        required
      />
      <button type="submit" data-testid="login-button" disabled={loading}>
        {loading ? 'Logging in...' : 'Login'}
      </button>
      {error && <div data-testid="auth-error">{error}</div>}
    </form>
  )
}

function TablesDashboard({ tables, onSelectTable, onCreateTable }) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newTableData, setNewTableData] = useState({ id: '', fields: [] })

  const handleCreateTable = async () => {
    try {
      await onCreateTable(newTableData)
      setShowCreateForm(false)
      setNewTableData({ id: '', fields: [] })
    } catch (error) {
      console.error('Failed to create table:', error)
      // In a real app, you'd show an error message to the user
    }
  }

  return (
    <div data-testid="tables-dashboard">
      <h2>Your Tables</h2>
      
      <button 
        data-testid="create-table-button"
        onClick={() => setShowCreateForm(true)}
      >
        Create New Table
      </button>

      {showCreateForm && (
        <div data-testid="create-table-form">
          <input
            data-testid="table-name-input"
            placeholder="Table name"
            value={newTableData.id}
            onChange={(e) => setNewTableData(prev => ({ ...prev, id: e.target.value }))}
          />
          <button data-testid="save-table-button" onClick={handleCreateTable}>
            Save Table
          </button>
          <button 
            data-testid="cancel-table-button"
            onClick={() => setShowCreateForm(false)}
          >
            Cancel
          </button>
        </div>
      )}

      <div data-testid="tables-list">
        {tables.map(table => (
          <div key={table.id} data-testid={`table-${table.id}`}>
            <span>{table.id}</span>
            <button 
              data-testid={`select-table-${table.id}`}
              onClick={() => onSelectTable(table.id)}
            >
              Open
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}

function TableView({ tableId, entities, onCreateEntity }) {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newEntityData, setNewEntityData] = useState({ name: '', email: '' })

  const handleCreateEntity = async () => {
    try {
      await onCreateEntity(newEntityData)
      setShowCreateForm(false)
      setNewEntityData({ name: '', email: '' })
    } catch (error) {
      console.error('Failed to create entity:', error)
      // In a real app, you'd show an error message to the user
    }
  }

  return (
    <div data-testid="table-view-content">
      <h3>Entities in {tableId}</h3>
      
      <button 
        data-testid="create-entity-button"
        onClick={() => setShowCreateForm(true)}
      >
        Add Entity
      </button>

      {showCreateForm && (
        <div data-testid="create-entity-form">
          <input
            data-testid="entity-name-input"
            placeholder="Name"
            value={newEntityData.name}
            onChange={(e) => setNewEntityData(prev => ({ ...prev, name: e.target.value }))}
          />
          <input
            data-testid="entity-email-input"
            placeholder="Email"
            value={newEntityData.email}
            onChange={(e) => setNewEntityData(prev => ({ ...prev, email: e.target.value }))}
          />
          <button data-testid="save-entity-button" onClick={handleCreateEntity}>
            Save Entity
          </button>
          <button 
            data-testid="cancel-entity-button"
            onClick={() => setShowCreateForm(false)}
          >
            Cancel
          </button>
        </div>
      )}

      <div data-testid="entities-list">
        {entities.map((entity, index) => (
          <div key={entity.id || index} data-testid={`entity-${entity.id || index}`}>
            <span>{entity.name} - {entity.email}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

import { useState } from 'react'

// Helper function to render app with providers
const renderApp = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <TestApp />
      </MantineProvider>
    </QueryClientProvider>
  )
}

describe('App Integration Tests', () => {
  const user = userEvent.setup()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Authentication Flow', () => {
    it('shows login form initially', () => {
      renderApp()
      
      expect(screen.getByTestId('auth-view')).toBeInTheDocument()
      expect(screen.getByTestId('login-form')).toBeInTheDocument()
    })

    it('handles successful login and shows dashboard', async () => {
      mockApiClient.login.mockResolvedValue({
        token: 'valid-token',
        user_id: 'testuser'
      })
      mockApiClient.getTables.mockResolvedValue({
        tables: [
          { id: 'contacts', fields: [] },
          { id: 'projects', fields: [] }
        ]
      })

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'testuser')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('login-button'))

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      })

      expect(screen.getByTestId('user-info')).toHaveTextContent('Welcome, testuser')
      expect(mockApiClient.login).toHaveBeenCalledWith('testuser', 'password123')
      expect(mockApiClient.getTables).toHaveBeenCalled()
    })

    it('handles login failure', async () => {
      mockApiClient.login.mockRejectedValue(new Error('Invalid credentials'))

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'wronguser')
      await user.type(screen.getByTestId('password-input'), 'wrongpass')
      await user.click(screen.getByTestId('login-button'))

      await waitFor(() => {
        expect(screen.getByTestId('auth-error')).toHaveTextContent('Invalid credentials')
      })

      expect(screen.getByTestId('auth-view')).toBeInTheDocument()
    })
  })

  describe('Dashboard Operations', () => {
    beforeEach(async () => {
      // Setup authenticated state
      mockApiClient.login.mockResolvedValue({
        token: 'valid-token',
        user_id: 'testuser'
      })
      mockApiClient.getTables.mockResolvedValue({
        tables: [
          { id: 'contacts', fields: [] }
        ]
      })

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'testuser')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('login-button'))

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      })
    })

    it('displays existing tables', () => {
      expect(screen.getByTestId('table-contacts')).toBeInTheDocument()
      expect(screen.getByText('contacts')).toBeInTheDocument()
    })

    it('can create a new table', async () => {
      mockApiClient.createTable.mockResolvedValue({
        id: 'newtable',
        fields: [],
        created_at: '2024-01-01T00:00:00Z'
      })

      await user.click(screen.getByTestId('create-table-button'))
      
      expect(screen.getByTestId('create-table-form')).toBeInTheDocument()

      await user.type(screen.getByTestId('table-name-input'), 'newtable')
      await user.click(screen.getByTestId('save-table-button'))

      await waitFor(() => {
        expect(mockApiClient.createTable).toHaveBeenCalledWith({
          id: 'newtable',
          fields: []
        })
      })

      expect(screen.getByTestId('table-newtable')).toBeInTheDocument()
    })

    it('can navigate to table view', async () => {
      mockApiClient.getEntities.mockResolvedValue({
        entities: [
          { id: '1', name: 'John Doe', email: 'john@example.com' }
        ]
      })

      await user.click(screen.getByTestId('select-table-contacts'))

      await waitFor(() => {
        expect(screen.getByTestId('table-view')).toBeInTheDocument()
      })

      expect(screen.getByTestId('table-header')).toHaveTextContent('Table: contacts')
      expect(mockApiClient.getEntities).toHaveBeenCalledWith('contacts')
    })

    it('can logout and return to auth view', async () => {
      await user.click(screen.getByTestId('logout-button'))

      expect(screen.getByTestId('auth-view')).toBeInTheDocument()
      expect(screen.queryByTestId('dashboard-view')).not.toBeInTheDocument()
    })
  })

  describe('Table View Operations', () => {
    beforeEach(async () => {
      // Setup authenticated state and navigate to table view
      mockApiClient.login.mockResolvedValue({
        token: 'valid-token',
        user_id: 'testuser'
      })
      mockApiClient.getTables.mockResolvedValue({
        tables: [{ id: 'contacts', fields: [] }]
      })
      mockApiClient.getEntities.mockResolvedValue({
        entities: [
          { id: '1', name: 'John Doe', email: 'john@example.com' }
        ]
      })

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'testuser')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('login-button'))

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      })

      await user.click(screen.getByTestId('select-table-contacts'))

      await waitFor(() => {
        expect(screen.getByTestId('table-view')).toBeInTheDocument()
      })
    })

    it('displays existing entities', () => {
      expect(screen.getByTestId('entity-1')).toBeInTheDocument()
      expect(screen.getByText('John Doe - john@example.com')).toBeInTheDocument()
    })

    it('can create a new entity', async () => {
      mockApiClient.createEntity.mockResolvedValue({
        id: '2',
        name: 'Jane Smith',
        email: 'jane@example.com'
      })

      await user.click(screen.getByTestId('create-entity-button'))
      
      expect(screen.getByTestId('create-entity-form')).toBeInTheDocument()

      await user.type(screen.getByTestId('entity-name-input'), 'Jane Smith')
      await user.type(screen.getByTestId('entity-email-input'), 'jane@example.com')
      await user.click(screen.getByTestId('save-entity-button'))

      await waitFor(() => {
        expect(mockApiClient.createEntity).toHaveBeenCalledWith('contacts', {
          name: 'Jane Smith',
          email: 'jane@example.com'
        })
      })

      expect(screen.getByTestId('entity-2')).toBeInTheDocument()
    })

    it('can navigate back to dashboard', async () => {
      await user.click(screen.getByTestId('back-button'))

      expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      expect(screen.queryByTestId('table-view')).not.toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('handles API errors gracefully during table creation', async () => {
      // Setup authenticated state
      mockApiClient.login.mockResolvedValue({
        token: 'valid-token',
        user_id: 'testuser'
      })
      mockApiClient.getTables.mockResolvedValue({ tables: [] })

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'testuser')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('login-button'))

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      })

      // Mock API error
      mockApiClient.createTable.mockRejectedValue(new Error('Table already exists'))

      await user.click(screen.getByTestId('create-table-button'))
      await user.type(screen.getByTestId('table-name-input'), 'existing-table')
      
      // In a real app, this would show an error message
      // For this test, we just verify the API was called
      await user.click(screen.getByTestId('save-table-button'))

      await waitFor(() => {
        expect(mockApiClient.createTable).toHaveBeenCalled()
      })
    })
  })

  describe('Loading States', () => {
    it('shows loading state during login', async () => {
      mockApiClient.login.mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve({
          token: 'token', user_id: 'user'
        }), 100))
      )
      mockApiClient.getTables.mockResolvedValue({ tables: [] })

      renderApp()

      await user.type(screen.getByTestId('user-id-input'), 'testuser')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('login-button'))

      expect(screen.getByTestId('login-button')).toHaveTextContent('Logging in...')
      expect(screen.getByTestId('login-button')).toBeDisabled()

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-view')).toBeInTheDocument()
      })
    })
  })
})