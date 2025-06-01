import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import App from './App';

// Mock the API service
vi.mock('./services/api', () => ({
  default: {
    getToken: vi.fn(() => null),
    getUserInfo: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    isAuthenticated: vi.fn(() => false),
  },
}));

// Mock notifications
vi.mock('@mantine/notifications', () => ({
  Notifications: () => null,
  notifications: {
    show: vi.fn(),
  },
}));

const renderApp = () => {
  return render(
    <MantineProvider>
      <App />
    </MantineProvider>
  );
};

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing', () => {
    renderApp();
    expect(document.body).toBeInTheDocument();
  });

  it('shows login page when not authenticated', async () => {
    renderApp();
    
    await waitFor(() => {
      expect(screen.getByText('Welcome to Notably')).toBeInTheDocument();
    });
  });

  it('shows sign in button', async () => {
    renderApp();
    
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument();
    });
  });

  it('has user id and password fields', async () => {
    renderApp();
    
    await waitFor(() => {
      expect(screen.getByLabelText(/user id/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });
  });
});