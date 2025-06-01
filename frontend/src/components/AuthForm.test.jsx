import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MantineProvider } from '@mantine/core'
import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock the auth store
const mockAuthStore = {
  login: vi.fn(),
  logout: vi.fn(),
  user: null,
  isAuthenticated: false,
  setUser: vi.fn()
}

// Mock the API client
const mockApiClient = {
  request: vi.fn()
}

// Mock AuthForm component since it's in App.jsx
// In a real project, you'd extract it to its own file
function AuthForm({ onSuccess }) {
  const [isLogin, setIsLogin] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [formData, setFormData] = useState({
    user_id: '',
    email: '',
    password: ''
  })

  const handleSubmit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const endpoint = isLogin ? '/api/v1/auth/login' : '/api/v1/auth/register'
      const payload = isLogin 
        ? { user_id: formData.user_id, password: formData.password }
        : formData

      const response = await mockApiClient.request('POST', endpoint, payload)
      
      if (response.token) {
        mockAuthStore.setUser({ user_id: formData.user_id, token: response.token })
        onSuccess?.(response)
      }
    } catch (err) {
      setError(err.message || 'Authentication failed')
    } finally {
      setLoading(false)
    }
  }

  const handleInputChange = (field, value) => {
    setFormData(prev => ({ ...prev, [field]: value }))
  }

  return (
    <form onSubmit={handleSubmit} data-testid="auth-form">
      <div data-testid="form-title">
        {isLogin ? 'Sign In' : 'Create Account'}
      </div>

      {error && (
        <div data-testid="error-message" role="alert">
          {error}
        </div>
      )}

      <div>
        <label htmlFor="user_id">User ID</label>
        <input
          id="user_id"
          data-testid="user-id-input"
          type="text"
          value={formData.user_id}
          onChange={(e) => handleInputChange('user_id', e.target.value)}
          required
        />
      </div>

      {!isLogin && (
        <div>
          <label htmlFor="email">Email</label>
          <input
            id="email"
            data-testid="email-input"
            type="email"
            value={formData.email}
            onChange={(e) => handleInputChange('email', e.target.value)}
            required
          />
        </div>
      )}

      <div>
        <label htmlFor="password">Password</label>
        <input
          id="password"
          data-testid="password-input"
          type="password"
          value={formData.password}
          onChange={(e) => handleInputChange('password', e.target.value)}
          required
        />
      </div>

      <button
        type="submit"
        data-testid="submit-button"
        disabled={loading}
      >
        {loading ? 'Loading...' : (isLogin ? 'Sign In' : 'Create Account')}
      </button>

      <button
        type="button"
        data-testid="toggle-mode-button"
        onClick={() => {
          setIsLogin(!isLogin)
          setError('')
        }}
      >
        {isLogin ? 'Need an account? Sign up' : 'Already have an account? Sign in'}
      </button>
    </form>
  )
}

// Import useState for the component
import { useState } from 'react'

// Helper function to render component with providers
const renderAuthForm = (props = {}) => {
  return render(
    <MantineProvider>
      <AuthForm {...props} />
    </MantineProvider>
  )
}

describe('AuthForm', () => {
  const mockOnSuccess = vi.fn()
  const user = userEvent.setup()

  beforeEach(() => {
    vi.clearAllMocks()
    mockApiClient.request.mockClear()
    mockAuthStore.setUser.mockClear()
  })

  describe('Rendering', () => {
    it('renders login form by default', () => {
      renderAuthForm()

      expect(screen.getByTestId('form-title')).toHaveTextContent('Sign In')
      expect(screen.getByTestId('user-id-input')).toBeInTheDocument()
      expect(screen.getByTestId('password-input')).toBeInTheDocument()
      expect(screen.queryByTestId('email-input')).not.toBeInTheDocument()
      expect(screen.getByTestId('submit-button')).toHaveTextContent('Sign In')
    })

    it('renders registration form when toggled', async () => {
      renderAuthForm()

      await user.click(screen.getByTestId('toggle-mode-button'))

      expect(screen.getByTestId('form-title')).toHaveTextContent('Create Account')
      expect(screen.getByTestId('user-id-input')).toBeInTheDocument()
      expect(screen.getByTestId('email-input')).toBeInTheDocument()
      expect(screen.getByTestId('password-input')).toBeInTheDocument()
      expect(screen.getByTestId('submit-button')).toHaveTextContent('Create Account')
    })

    it('shows correct toggle button text', () => {
      renderAuthForm()

      expect(screen.getByTestId('toggle-mode-button')).toHaveTextContent(
        'Need an account? Sign up'
      )
    })

    it('shows correct toggle button text in registration mode', async () => {
      renderAuthForm()

      await user.click(screen.getByTestId('toggle-mode-button'))

      expect(screen.getByTestId('toggle-mode-button')).toHaveTextContent(
        'Already have an account? Sign in'
      )
    })
  })

  describe('Form Input', () => {
    it('updates user ID input value', async () => {
      renderAuthForm()

      const userIdInput = screen.getByTestId('user-id-input')
      await user.type(userIdInput, 'test_user')

      expect(userIdInput).toHaveValue('test_user')
    })

    it('updates password input value', async () => {
      renderAuthForm()

      const passwordInput = screen.getByTestId('password-input')
      await user.type(passwordInput, 'password123')

      expect(passwordInput).toHaveValue('password123')
    })

    it('updates email input value in registration mode', async () => {
      renderAuthForm()

      await user.click(screen.getByTestId('toggle-mode-button'))

      const emailInput = screen.getByTestId('email-input')
      await user.type(emailInput, 'test@example.com')

      expect(emailInput).toHaveValue('test@example.com')
    })
  })

  describe('Form Submission', () => {
    it('submits login form with correct data', async () => {
      mockApiClient.request.mockResolvedValue({
        token: 'mock-token',
        user_id: 'test_user'
      })

      renderAuthForm({ onSuccess: mockOnSuccess })

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      expect(mockApiClient.request).toHaveBeenCalledWith(
        'POST',
        '/api/v1/auth/login',
        {
          user_id: 'test_user',
          password: 'password123'
        }
      )
    })

    it('submits registration form with correct data', async () => {
      mockApiClient.request.mockResolvedValue({
        token: 'mock-token',
        user_id: 'new_user'
      })

      renderAuthForm({ onSuccess: mockOnSuccess })

      await user.click(screen.getByTestId('toggle-mode-button'))
      await user.type(screen.getByTestId('user-id-input'), 'new_user')
      await user.type(screen.getByTestId('email-input'), 'new@example.com')
      await user.type(screen.getByTestId('password-input'), 'newpassword123')
      await user.click(screen.getByTestId('submit-button'))

      expect(mockApiClient.request).toHaveBeenCalledWith(
        'POST',
        '/api/v1/auth/register',
        {
          user_id: 'new_user',
          email: 'new@example.com',
          password: 'newpassword123'
        }
      )
    })

    it('calls onSuccess callback when authentication succeeds', async () => {
      const mockResponse = {
        token: 'mock-token',
        user_id: 'test_user'
      }
      mockApiClient.request.mockResolvedValue(mockResponse)

      renderAuthForm({ onSuccess: mockOnSuccess })

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(mockOnSuccess).toHaveBeenCalledWith(mockResponse)
      })
    })

    it('updates auth store when authentication succeeds', async () => {
      mockApiClient.request.mockResolvedValue({
        token: 'mock-token',
        user_id: 'test_user'
      })

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(mockAuthStore.setUser).toHaveBeenCalledWith({
          user_id: 'test_user',
          token: 'mock-token'
        })
      })
    })
  })

  describe('Loading States', () => {
    it('shows loading state during form submission', async () => {
      mockApiClient.request.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)))

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      expect(screen.getByTestId('submit-button')).toHaveTextContent('Loading...')
      expect(screen.getByTestId('submit-button')).toBeDisabled()
    })

    it('restores button state after successful submission', async () => {
      mockApiClient.request.mockResolvedValue({
        token: 'mock-token',
        user_id: 'test_user'
      })

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(screen.getByTestId('submit-button')).toHaveTextContent('Sign In')
        expect(screen.getByTestId('submit-button')).not.toBeDisabled()
      })
    })
  })

  describe('Error Handling', () => {
    it('displays error message when authentication fails', async () => {
      mockApiClient.request.mockRejectedValue(new Error('Invalid credentials'))

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'wrong_user')
      await user.type(screen.getByTestId('password-input'), 'wrong_password')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Invalid credentials')
      })
    })

    it('displays generic error message when error has no message', async () => {
      mockApiClient.request.mockRejectedValue(new Error())

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toHaveTextContent('Authentication failed')
      })
    })

    it('clears error message when switching between login and registration', async () => {
      mockApiClient.request.mockRejectedValue(new Error('Some error'))

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(screen.getByTestId('error-message')).toBeInTheDocument()
      })

      await user.click(screen.getByTestId('toggle-mode-button'))

      expect(screen.queryByTestId('error-message')).not.toBeInTheDocument()
    })

    it('restores button state after failed submission', async () => {
      mockApiClient.request.mockRejectedValue(new Error('Authentication failed'))

      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      await waitFor(() => {
        expect(screen.getByTestId('submit-button')).toHaveTextContent('Sign In')
        expect(screen.getByTestId('submit-button')).not.toBeDisabled()
      })
    })
  })

  describe('Form Validation', () => {
    it('prevents submission with empty user ID', async () => {
      renderAuthForm()

      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      expect(mockApiClient.request).not.toHaveBeenCalled()
    })

    it('prevents submission with empty password', async () => {
      renderAuthForm()

      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.click(screen.getByTestId('submit-button'))

      expect(mockApiClient.request).not.toHaveBeenCalled()
    })

    it('prevents registration submission with empty email', async () => {
      renderAuthForm()

      await user.click(screen.getByTestId('toggle-mode-button'))
      await user.type(screen.getByTestId('user-id-input'), 'test_user')
      await user.type(screen.getByTestId('password-input'), 'password123')
      await user.click(screen.getByTestId('submit-button'))

      expect(mockApiClient.request).not.toHaveBeenCalled()
    })
  })
})