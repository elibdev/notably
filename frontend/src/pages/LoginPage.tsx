import React, { useState } from 'react';
import {
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Anchor,
  Stack,
  Container,
  Alert,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { IconAlertCircle } from '@tabler/icons-react';
import { Link, Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { LoginRequest, ApiError } from '../types/api';

const LoginPage: React.FC = () => {
  const { login, isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<LoginRequest>({
    initialValues: {
      user_id: '',
      password: '',
    },
    validate: {
      user_id: (value) => {
        if (!value.trim()) return 'User ID is required';
        if (value.length < 3) return 'User ID must be at least 3 characters';
        return null;
      },
      password: (value) => {
        if (!value) return 'Password is required';
        if (value.length < 6) return 'Password must be at least 6 characters';
        return null;
      },
    },
  });

  // Redirect if already authenticated
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  const handleSubmit = async (values: LoginRequest) => {
    setLoading(true);
    setError(null);

    try {
      await login(values);
      // Navigation will happen automatically via AuthContext
    } catch (err) {
      const apiError = err as ApiError;
      setError(apiError.message || 'Login failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size={420} my={40}>
      <Title ta="center" mb="md">
        Welcome to Notably
      </Title>
      
      <Text c="dimmed" size="sm" ta="center" mb="xl">
        Sign in to manage your data tables
      </Text>

      <Paper withBorder shadow="md" p="xl" radius="md">
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            {error && (
              <Alert
                icon={<IconAlertCircle size="1rem" />}
                color="red"
                variant="light"
              >
                {error}
              </Alert>
            )}

            <TextInput
              label="User ID"
              placeholder="Enter your user ID"
              required
              {...form.getInputProps('user_id')}
              disabled={loading}
            />

            <PasswordInput
              label="Password"
              placeholder="Enter your password"
              required
              {...form.getInputProps('password')}
              disabled={loading}
            />

            <Button
              type="submit"
              fullWidth
              loading={loading}
              mt="md"
            >
              Sign In
            </Button>

            <Text ta="center" mt="md" size="sm">
              Don't have an account?{' '}
              <Anchor component={Link} to="/register" size="sm">
                Create account
              </Anchor>
            </Text>
          </Stack>
        </form>
      </Paper>
    </Container>
  );
};

export default LoginPage;