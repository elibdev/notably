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
import { RegisterRequest, ApiError } from '../types/api';

const RegisterPage: React.FC = () => {
  const { register, isAuthenticated } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const form = useForm<RegisterRequest>({
    initialValues: {
      user_id: '',
      email: '',
      password: '',
    },
    validate: {
      user_id: (value) => {
        if (!value.trim()) return 'User ID is required';
        if (value.length < 3) return 'User ID must be at least 3 characters';
        if (!/^[a-zA-Z0-9_-]+$/.test(value)) return 'User ID can only contain letters, numbers, underscores, and hyphens';
        return null;
      },
      email: (value) => {
        if (!value.trim()) return 'Email is required';
        if (!/^\S+@\S+\.\S+$/.test(value)) return 'Invalid email format';
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

  const handleSubmit = async (values: RegisterRequest) => {
    setLoading(true);
    setError(null);

    try {
      await register(values);
      // Navigation will happen automatically via AuthContext
    } catch (err) {
      const apiError = err as ApiError;
      setError(apiError.message || 'Registration failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size={420} my={40}>
      <Title ta="center" mb="md">
        Create Account
      </Title>
      
      <Text c="dimmed" size="sm" ta="center" mb="xl">
        Sign up to start using Notably
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
              placeholder="Choose a unique user ID"
              required
              {...form.getInputProps('user_id')}
              disabled={loading}
            />

            <TextInput
              label="Email"
              placeholder="Enter your email address"
              type="email"
              required
              {...form.getInputProps('email')}
              disabled={loading}
            />

            <PasswordInput
              label="Password"
              placeholder="Choose a strong password"
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
              Create Account
            </Button>

            <Text ta="center" mt="md" size="sm">
              Already have an account?{' '}
              <Anchor component={Link} to="/login" size="sm">
                Sign in
              </Anchor>
            </Text>
          </Stack>
        </form>
      </Paper>
    </Container>
  );
};

export default RegisterPage;