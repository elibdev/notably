import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { notifications } from '@mantine/notifications';
import apiService from '../services/api';
import { 
  AuthContext as AuthContextType, 
  LoginRequest, 
  RegisterRequest, 
  UserInfoResponse,
  ApiError 
} from '../types/api';

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<UserInfoResponse | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Initialize auth state on mount
  useEffect(() => {
    const initializeAuth = async () => {
      const storedToken = apiService.getToken();
      
      if (storedToken) {
        try {
          // Verify token is still valid by fetching user info
          const userInfo = await apiService.getUserInfo();
          setUser(userInfo);
          setToken(storedToken);
          setIsAuthenticated(true);
        } catch (error) {
          // Token is invalid, clear it
          apiService.logout();
          localStorage.removeItem('auth_token');
          console.error('Token validation failed:', error);
        }
      }
      
      setIsLoading(false);
    };

    initializeAuth();
  }, []);

  const login = async (credentials: LoginRequest): Promise<void> => {
    try {
      const authResponse = await apiService.login(credentials);
      const userInfo = await apiService.getUserInfo();
      
      setUser(userInfo);
      setToken(authResponse.token);
      setIsAuthenticated(true);
      
      notifications.show({
        title: 'Login Successful',
        message: `Welcome back, ${userInfo.user_id}!`,
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Login Failed',
        message: apiError.message || 'Invalid credentials',
        color: 'red',
      });
      throw error;
    }
  };

  const register = async (data: RegisterRequest): Promise<void> => {
    try {
      const authResponse = await apiService.register(data);
      const userInfo = await apiService.getUserInfo();
      
      setUser(userInfo);
      setToken(authResponse.token);
      setIsAuthenticated(true);
      
      notifications.show({
        title: 'Registration Successful',
        message: `Welcome to Notably, ${userInfo.user_id}!`,
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Registration Failed',
        message: apiError.message || 'Registration failed',
        color: 'red',
      });
      throw error;
    }
  };

  const logout = (): void => {
    apiService.logout();
    setUser(null);
    setToken(null);
    setIsAuthenticated(false);
    
    notifications.show({
      title: 'Logged Out',
      message: 'You have been successfully logged out',
      color: 'blue',
    });
  };

  const contextValue: AuthContextType = {
    user,
    token,
    isAuthenticated,
    login,
    register,
    logout,
  };

  if (isLoading) {
    return null; // or a loading spinner
  }

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

export default AuthContext;