#!/bin/bash

echo "🚀 Setting up Notably Frontend (React + Vite + Mantine)..."
echo "This script will create everything you need!"
echo ""

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    echo "Visit: https://nodejs.org/"
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 18 ]; then
    echo "❌ Node.js version must be 18 or higher. Current version: $(node --version)"
    echo "Please update Node.js: https://nodejs.org/"
    exit 1
fi

echo "✅ Node.js $(node --version) detected"

# Create project directory
PROJECT_NAME="notably-frontend"
if [ -d "$PROJECT_NAME" ]; then
    echo "⚠️  Directory $PROJECT_NAME already exists!"
    read -p "Do you want to remove it and continue? (y/N): " confirm
    if [[ $confirm == [yY] || $confirm == [yY][eE][sS] ]]; then
        rm -rf "$PROJECT_NAME"
        echo "🗑️  Removed existing directory"
    else
        echo "❌ Setup cancelled"
        exit 1
    fi
fi

echo "📦 Creating Vite project..."
npm create vite@latest $PROJECT_NAME -- --template react

cd $PROJECT_NAME

echo "📚 Installing dependencies..."
npm install @mantine/core @mantine/hooks @mantine/form @mantine/notifications @mantine/dates @mantine/modals @tabler/icons-react dayjs @tanstack/react-query zustand

echo "📝 Creating Notably-specific files..."

# Create vite.config.js
cat > vite.config.js << 'EOF'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      }
    }
  }
})
EOF

# Update package.json with proper dependencies
cat > package.json << 'EOF'
{
  "name": "notably-frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "lint": "eslint . --ext js,jsx --report-unused-disable-directives --max-warnings 0",
    "preview": "vite preview"
  },
  "dependencies": {
    "@mantine/core": "^7.10.0",
    "@mantine/dates": "^7.10.0",
    "@mantine/form": "^7.10.0",
    "@mantine/hooks": "^7.10.0",
    "@mantine/modals": "^7.10.0",
    "@mantine/notifications": "^7.10.0",
    "@tabler/icons-react": "^3.5.0",
    "@tanstack/react-query": "^4.32.0",
    "dayjs": "^1.11.9",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "zustand": "^4.4.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.15",
    "@types/react-dom": "^18.2.7",
    "@vitejs/plugin-react": "^4.0.3",
    "eslint": "^8.45.0",
    "eslint-plugin-react": "^7.32.2",
    "eslint-plugin-react-hooks": "^4.6.0",
    "eslint-plugin-react-refresh": "^0.4.3",
    "vite": "^4.4.5"
  }
}
EOF

# Create main.jsx
cat > src/main.jsx << 'EOF'
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'
import '@mantine/core/styles.css'
import '@mantine/dates/styles.css'
import '@mantine/notifications/styles.css'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
EOF

# Update index.html
cat > index.html << 'EOF'
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Notably - Time-Series Database Platform</title>
    <meta name="description" content="Modern time-series database with audit trails and time travel capabilities" />
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>
EOF

echo "🎨 Creating the main App component..."

# Create the main App.jsx file (this is the big one!)
cat > src/App.jsx << 'EOF'
import React from 'react';
import {
  MantineProvider,
  AppShell,
  Header,
  Navbar,
  Text,
  Group,
  ActionIcon,
  Button,
  Avatar,
  Menu,
  Container,
  Paper,
  Title,
  Badge,
  Card,
  Grid,
  Stack,
  Tabs,
  TextInput,
  PasswordInput,
  Select,
  NumberInput,
  Textarea,
  Switch,
  Modal,
  Table,
  Anchor,
  Breadcrumbs,
  Loader,
  Center,
  Box,
  ScrollArea,
  Timeline,
  Code,
  JsonInput,
  Spotlight,
  rem,
  useMantineTheme,
} from '@mantine/core';
import { DateTimePicker } from '@mantine/dates';
import { useForm } from '@mantine/form';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { modals } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { ModalsProvider } from '@mantine/modals';
import { QueryClient, QueryClientProvider, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  IconDatabase,
  IconTable,
  IconPlus,
  IconEdit,
  IconTrash,
  IconHistory,
  IconUser,
  IconLogout,
  IconSearch,
  IconClock,
  IconRefresh,
  IconChevronRight,
  IconHome,
  IconSettings,
  IconBolt,
  IconActivity,
  IconRestore,
} from '@tabler/icons-react';
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import dayjs from 'dayjs';

// Create a query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      retry: 2,
    },
  },
});

// Zustand store for auth state
const useAuthStore = create(
  persist(
    (set, get) => ({
      token: null,
      user: null,
      login: (token, user) => set({ token, user }),
      logout: () => set({ token: null, user: null }),
      isAuthenticated: () => !!get().token,
    }),
    {
      name: 'notably-auth',
    }
  )
);

// API utility functions
const apiClient = {
  async request(endpoint, options = {}) {
    const { token } = useAuthStore.getState();

    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
      },
      ...options,
    };

    const response = await fetch(`/api/v1${endpoint}`, config);

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    return null;
  },

  get: (endpoint) => apiClient.request(endpoint),
  post: (endpoint, data) => apiClient.request(endpoint, { method: 'POST', body: JSON.stringify(data) }),
  put: (endpoint, data) => apiClient.request(endpoint, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (endpoint) => apiClient.request(endpoint, { method: 'DELETE' }),
};

// Authentication component
function AuthForm() {
  const theme = useMantineTheme();
  const [activeTab, setActiveTab] = React.useState('login');
  const { login } = useAuthStore();

  const loginForm = useForm({
    initialValues: { user_id: '', password: '' },
    validate: {
      user_id: (value) => (!value ? 'User ID is required' : null),
      password: (value) => (!value ? 'Password is required' : null),
    },
  });

  const registerForm = useForm({
    initialValues: { user_id: '', email: '', password: '' },
    validate: {
      user_id: (value) => (!value ? 'User ID is required' : null),
      email: (value) => (!/^\S+@\S+$/.test(value) ? 'Invalid email' : null),
      password: (value) => (value.length < 8 ? 'Password must be at least 8 characters' : null),
    },
  });

  const loginMutation = useMutation({
    mutationFn: async (data) => await apiClient.post('/auth/login', data),
    onSuccess: (response) => {
      login(response.token, response.user_id);
      notifications.show({
        title: 'Welcome back!',
        message: 'Successfully logged in',
        color: 'green',
      });
    },
    onError: (error) => {
      notifications.show({
        title: 'Login failed',
        message: error.message,
        color: 'red',
      });
    },
  });

  const registerMutation = useMutation({
    mutationFn: async (data) => await apiClient.post('/auth/register', data),
    onSuccess: (response) => {
      login(response.token, response.user_id);
      notifications.show({
        title: 'Welcome to Notably!',
        message: 'Account created successfully',
        color: 'green',
      });
    },
    onError: (error) => {
      notifications.show({
        title: 'Registration failed',
        message: error.message,
        color: 'red',
      });
    },
  });

  return (
    <Container size={420} my={80}>
      <Paper withBorder shadow="xl" p={30} mt={30} radius="xl">
        <Group justify="center" mb="xl">
          <IconDatabase size={48} color={theme.colors.blue[6]} />
          <Stack gap={0}>
            <Title order={1} size="h2" fw={900} c="blue">
              Notably
            </Title>
            <Text c="dimmed" size="sm">
              Time-Series Database Platform
            </Text>
          </Stack>
        </Group>

        <Tabs value={activeTab} onChange={setActiveTab} variant="pills" radius="xl">
          <Tabs.List grow mb="xl">
            <Tabs.Tab value="login">Login</Tabs.Tab>
            <Tabs.Tab value="register">Register</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="login">
            <form onSubmit={loginForm.onSubmit((values) => loginMutation.mutate(values))}>
              <Stack>
                <TextInput
                  label="User ID"
                  placeholder="your_user_id"
                  {...loginForm.getInputProps('user_id')}
                  leftSection={<IconUser size={16} />}
                />
                <PasswordInput
                  label="Password"
                  placeholder="Your password"
                  {...loginForm.getInputProps('password')}
                />
                <Button
                  type="submit"
                  fullWidth
                  loading={loginMutation.isPending}
                  leftSection={<IconBolt size={16} />}
                  radius="xl"
                  size="lg"
                  variant="gradient"
                  gradient={{ from: 'blue', to: 'cyan' }}
                >
                  Login
                </Button>
              </Stack>
            </form>
          </Tabs.Panel>

          <Tabs.Panel value="register">
            <form onSubmit={registerForm.onSubmit((values) => registerMutation.mutate(values))}>
              <Stack>
                <TextInput
                  label="User ID"
                  placeholder="your_user_id"
                  description="3-50 characters, alphanumeric and underscores"
                  {...registerForm.getInputProps('user_id')}
                  leftSection={<IconUser size={16} />}
                />
                <TextInput
                  label="Email"
                  placeholder="your@email.com"
                  {...registerForm.getInputProps('email')}
                />
                <PasswordInput
                  label="Password"
                  placeholder="Your password"
                  description="Minimum 8 characters"
                  {...registerForm.getInputProps('password')}
                />
                <Button
                  type="submit"
                  fullWidth
                  loading={registerMutation.isPending}
                  leftSection={<IconPlus size={16} />}
                  radius="xl"
                  size="lg"
                  variant="gradient"
                  gradient={{ from: 'indigo', to: 'cyan' }}
                >
                  Create Account
                </Button>
              </Stack>
            </form>
          </Tabs.Panel>
        </Tabs>
      </Paper>
    </Container>
  );
}

// Table card component
function TableCard({ table, onSelect, onShowHistory }) {
  const theme = useMantineTheme();

  return (
    <Card
      shadow="lg"
      radius="xl"
      withBorder
      style={{
        cursor: 'pointer',
        transition: 'all 0.2s ease',
        borderLeft: `4px solid ${theme.colors.blue[6]}`,
      }}
      onClick={() => onSelect(table.id)}
    >
      <Group justify="space-between" mb="xs">
        <Group>
          <IconTable size={24} color={theme.colors.blue[6]} />
          <Text fw={600} size="lg">
            {table.id}
          </Text>
        </Group>
        <Badge variant="light" color="blue">
          {table.fields.length} field{table.fields.length !== 1 ? 's' : ''}
        </Badge>
      </Group>

      <Text size="sm" c="dimmed" mb="md">
        {table.fields.map(field => `${field.name}: ${field.data_type}`).join(', ')}
      </Text>

      <Group justify="space-between">
        <Text size="xs" c="dimmed">
          <IconClock size={14} style={{ marginRight: 4 }} />
          {dayjs(table.created_at).format('MMM D, YYYY')}
        </Text>
        <Group gap="xs">
          <Button
            variant="light"
            size="xs"
            leftSection={<IconEdit size={12} />}
            onClick={(e) => {
              e.stopPropagation();
              onSelect(table.id);
            }}
          >
            View
          </Button>
          <Button
            variant="subtle"
            size="xs"
            leftSection={<IconHistory size={12} />}
            onClick={(e) => {
              e.stopPropagation();
              onShowHistory(table.id);
            }}
          >
            History
          </Button>
        </Group>
      </Group>
    </Card>
  );
}

// Tables dashboard
function TablesDashboard({ onSelectTable }) {
  const [createTableOpened, { open: openCreateTable, close: closeCreateTable }] = useDisclosure(false);

  const { data: tables = [], isLoading, error } = useQuery({
    queryKey: ['tables'],
    queryFn: () => apiClient.get('/tables'),
  });

  const queryClient = useQueryClient();

  const createTableMutation = useMutation({
    mutationFn: async (data) => await apiClient.post('/tables', data),
    onSuccess: () => {
      queryClient.invalidateQueries(['tables']);
      closeCreateTable();
      notifications.show({
        title: 'Success',
        message: 'Table created successfully',
        color: 'green',
      });
    },
    onError: (error) => {
      notifications.show({
        title: 'Error',
        message: error.message,
        color: 'red',
      });
    },
  });

  const createTableForm = useForm({
    initialValues: {
      id: '',
      fields: [{ name: '', data_type: '' }],
    },
    validate: {
      id: (value) => (!value ? 'Table ID is required' : null),
      fields: {
        name: (value) => (!value ? 'Field name is required' : null),
        data_type: (value) => (!value ? 'Data type is required' : null),
      },
    },
  });

  const showHistory = (tableId) => {
    // TODO: Implement table history modal
    notifications.show({
      title: 'Coming Soon',
      message: 'Table history view will be implemented',
      color: 'blue',
    });
  };

  if (isLoading) {
    return (
      <Center h={400}>
        <Stack align="center">
          <Loader size="xl" variant="bars" />
          <Text c="dimmed">Loading tables...</Text>
        </Stack>
      </Center>
    );
  }

  if (error) {
    return (
      <Center h={400}>
        <Stack align="center">
          <Text c="red">Failed to load tables</Text>
          <Button leftSection={<IconRefresh size={16} />} onClick={() => queryClient.invalidateQueries(['tables'])}>
            Retry
          </Button>
        </Stack>
      </Center>
    );
  }

  return (
    <>
      <Group justify="space-between" mb="xl">
        <Stack gap={0}>
          <Title order={1} size="h2">
            <IconTable style={{ marginRight: 8 }} />
            Tables
          </Title>
          <Text c="dimmed">Manage your data schemas and structures</Text>
        </Stack>
        <Button
          leftSection={<IconPlus size={16} />}
          onClick={openCreateTable}
          radius="xl"
          size="lg"
          variant="gradient"
          gradient={{ from: 'blue', to: 'cyan' }}
        >
          Create Table
        </Button>
      </Group>

      {tables.tables?.length === 0 ? (
        <Paper radius="xl" p="xl" withBorder>
          <Stack align="center" gap="xl">
            <IconDatabase size={80} color="gray" opacity={0.3} />
            <Stack align="center" gap="xs">
              <Title order={3} c="dimmed">
                No tables yet
              </Title>
              <Text c="dimmed" ta="center">
                Create your first table to get started with storing time-series data
              </Text>
            </Stack>
            <Button
              leftSection={<IconPlus size={16} />}
              onClick={openCreateTable}
              radius="xl"
              size="lg"
              variant="gradient"
              gradient={{ from: 'blue', to: 'cyan' }}
            >
              Create Your First Table
            </Button>
          </Stack>
        </Paper>
      ) : (
        <Grid>
          {tables.tables?.map((table) => (
            <Grid.Col key={table.id} span={{ base: 12, md: 6, lg: 4 }}>
              <TableCard table={table} onSelect={onSelectTable} onShowHistory={showHistory} />
            </Grid.Col>
          ))}
        </Grid>
      )}

      <Modal
        opened={createTableOpened}
        onClose={closeCreateTable}
        title={
          <Group>
            <IconTable size={20} />
            <Text fw={600}>Create New Table</Text>
          </Group>
        }
        size="lg"
        radius="xl"
      >
        <form
          onSubmit={createTableForm.onSubmit((values) => {
            const validFields = values.fields.filter((field) => field.name && field.data_type);
            createTableMutation.mutate({ ...values, fields: validFields });
          })}
        >
          <Stack>
            <TextInput
              label="Table ID"
              placeholder="users, products, orders..."
              description="Unique identifier for your table"
              {...createTableForm.getInputProps('id')}
            />

            <Stack gap="xs">
              <Text fw={500}>Fields</Text>
              {createTableForm.values.fields.map((field, index) => (
                <Group key={index} align="end">
                  <TextInput
                    placeholder="Field name"
                    style={{ flex: 1 }}
                    {...createTableForm.getInputProps(`fields.${index}.name`)}
                  />
                  <Select
                    placeholder="Type"
                    style={{ width: 120 }}
                    data={[
                      { value: 'string', label: 'String' },
                      { value: 'int', label: 'Integer' },
                      { value: 'float', label: 'Float' },
                      { value: 'bool', label: 'Boolean' },
                      { value: 'date', label: 'Date' },
                      { value: 'json', label: 'JSON' },
                      { value: 'reference', label: 'Reference' },
                    ]}
                    {...createTableForm.getInputProps(`fields.${index}.data_type`)}
                  />
                  <ActionIcon
                    color="red"
                    variant="light"
                    onClick={() => {
                      if (createTableForm.values.fields.length > 1) {
                        createTableForm.removeListItem('fields', index);
                      }
                    }}
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Group>
              ))}
              <Button
                variant="light"
                leftSection={<IconPlus size={16} />}
                onClick={() => createTableForm.insertListItem('fields', { name: '', data_type: '' })}
              >
                Add Field
              </Button>
            </Stack>

            <Group justify="flex-end" pt="md">
              <Button variant="subtle" onClick={closeCreateTable}>
                Cancel
              </Button>
              <Button type="submit" loading={createTableMutation.isPending} leftSection={<IconDatabase size={16} />}>
                Create Table
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </>
  );
}

// Entity row component
function EntityRow({ entity, table, onEdit, onDelete, onUndelete, onShowHistory }) {
  const theme = useMantineTheme();

  return (
    <Table.Tr key={entity.entity_id} style={{ opacity: entity.is_deleted ? 0.6 : 1 }}>
      <Table.Td>
        <Group gap="xs">
          <Code>{entity.entity_id.substring(0, 8)}...</Code>
          {entity.is_deleted && <Badge color="orange" size="xs">Deleted</Badge>}
        </Group>
      </Table.Td>
      {table.fields.map((field) => {
        const value = entity.fields[field.name];
        if (value === null || value === undefined) {
          return <Table.Td key={field.name}><Text c="dimmed">-</Text></Table.Td>;
        }

        if (field.data_type === 'json') {
          return (
            <Table.Td key={field.name}>
              <Code block>{JSON.stringify(value, null, 2)}</Code>
            </Table.Td>
          );
        }

        if (field.data_type === 'bool') {
          return (
            <Table.Td key={field.name}>
              <Badge color={value ? 'green' : 'gray'} variant="light">
                {value ? 'true' : 'false'}
              </Badge>
            </Table.Td>
          );
        }

        if (field.data_type === 'date') {
          return (
            <Table.Td key={field.name}>
              {dayjs(value).format('MMM D, YYYY HH:mm')}
            </Table.Td>
          );
        }

        return <Table.Td key={field.name}>{String(value)}</Table.Td>;
      })}
      <Table.Td>{dayjs(entity.timestamp).format('MMM D, YYYY HH:mm')}</Table.Td>
      <Table.Td>
        <Group gap="xs">
          <ActionIcon variant="light" onClick={() => onEdit(entity)}>
            <IconEdit size={16} />
          </ActionIcon>
          <ActionIcon variant="light" color="gray" onClick={() => onShowHistory(entity.entity_id)}>
            <IconHistory size={16} />
          </ActionIcon>
          {entity.is_deleted ? (
            <ActionIcon variant="light" color="green" onClick={() => onUndelete(entity.entity_id)}>
              <IconRestore size={16} />
            </ActionIcon>
          ) : (
            <ActionIcon variant="light" color="red" onClick={() => onDelete(entity.entity_id)}>
              <IconTrash size={16} />
            </ActionIcon>
          )}
        </Group>
      </Table.Td>
    </Table.Tr>
  );
}

// Table view component
function TableView({ tableId, onBack }) {
  const [entityModalOpened, { open: openEntityModal, close: closeEntityModal }] = useDisclosure(false);
  const [selectedEntity, setSelectedEntity] = React.useState(null);
  const [asOfDate, setAsOfDate] = React.useState(null);

  const { data: table } = useQuery({
    queryKey: ['table', tableId],
    queryFn: () => apiClient.get(`/tables/${tableId}`),
  });

  const entitiesQuery = useQuery({
    queryKey: ['entities', tableId, asOfDate],
    queryFn: () => {
      let endpoint = `/tables/${tableId}/entities?limit=100`;
      if (asOfDate) {
        endpoint += `&asOf=${asOfDate.toISOString()}`;
      }
      return apiClient.get(endpoint);
    },
  });

  const queryClient = useQueryClient();

  const createEntityMutation = useMutation({
    mutationFn: async (data) => await apiClient.post(`/tables/${tableId}/entities`, data),
    onSuccess: () => {
      queryClient.invalidateQueries(['entities', tableId]);
      closeEntityModal();
      notifications.show({ title: 'Success', message: 'Entity created', color: 'green' });
    },
  });

  const updateEntityMutation = useMutation({
    mutationFn: async ({ entityId, data }) => await apiClient.put(`/tables/${tableId}/entities/${entityId}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries(['entities', tableId]);
      closeEntityModal();
      notifications.show({ title: 'Success', message: 'Entity updated', color: 'green' });
    },
  });

  const deleteEntityMutation = useMutation({
    mutationFn: async (entityId) => await apiClient.delete(`/tables/${tableId}/entities/${entityId}`),
    onSuccess: () => {
      queryClient.invalidateQueries(['entities', tableId]);
      notifications.show({ title: 'Success', message: 'Entity deleted', color: 'green' });
    },
  });

  const undeleteEntityMutation = useMutation({
    mutationFn: async (entityId) => await apiClient.post(`/tables/${tableId}/entities/${entityId}/undelete`),
    onSuccess: () => {
      queryClient.invalidateQueries(['entities', tableId]);
      notifications.show({ title: 'Success', message: 'Entity restored', color: 'green' });
    },
  });

  const entityForm = useForm({
    initialValues: {},
  });

  const openCreateEntity = () => {
    setSelectedEntity(null);
    const initialValues = {};
    table?.fields.forEach((field) => {
      initialValues[field.name] = field.data_type === 'bool' ? false : '';
    });
    entityForm.setValues(initialValues);
    openEntityModal();
  };

  const openEditEntity = (entity) => {
    setSelectedEntity(entity);
    entityForm.setValues(entity.fields);
    openEntityModal();
  };

  const handleSubmit = (values) => {
    const fields = {};
    Object.entries(values).forEach(([key, value]) => {
      if (value !== '' && value !== null && value !== undefined) {
        fields[key] = value;
      }
    });

    if (selectedEntity) {
      updateEntityMutation.mutate({ entityId: selectedEntity.entity_id, data: { fields } });
    } else {
      createEntityMutation.mutate({ fields });
    }
  };

  const showHistory = (entityId) => {
    // TODO: Implement entity history modal
    notifications.show({
      title: 'Coming Soon',
      message: 'Entity history view will be implemented',
      color: 'blue',
    });
  };

  if (!table) {
    return (
      <Center h={400}>
        <Loader size="xl" />
      </Center>
    );
  }

  const entities = entitiesQuery.data?.entities || [];

  return (
    <>
      <Stack gap="xl">
        <Group justify="space-between">
          <Stack gap={0}>
            <Group>
              <ActionIcon variant="light" onClick={onBack}>
                <IconChevronRight style={{ transform: 'rotate(180deg)' }} size={16} />
              </ActionIcon>
              <Title order={1} size="h2">
                <IconTable style={{ marginRight: 8 }} />
                {tableId}
              </Title>
            </Group>
            <Text c="dimmed">Manage entities in this table</Text>
          </Stack>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={openCreateEntity}
            radius="xl"
            variant="gradient"
            gradient={{ from: 'blue', to: 'cyan' }}
          >
            Add Entity
          </Button>
        </Group>

        <Paper withBorder radius="xl" p="lg" style={{ background: 'linear-gradient(45deg, #f8f9fa, #e9ecef)' }}>
          <Group justify="space-between" mb="md">
            <Group>
              <IconClock size={20} />
              <Text fw={600}>Time Travel</Text>
            </Group>
            <Badge variant="light" leftSection={<IconActivity size={12} />}>
              {asOfDate ? 'Historical View' : 'Current State'}
            </Badge>
          </Group>
          <Group>
            <DateTimePicker
              label="View data as of"
              placeholder="Select date and time"
              value={asOfDate}
              onChange={setAsOfDate}
              clearable
              style={{ flex: 1 }}
            />
            <Button
              variant="light"
              leftSection={<IconSearch size={16} />}
              onClick={() => entitiesQuery.refetch()}
              loading={entitiesQuery.isLoading}
            >
              Query
            </Button>
          </Group>
        </Paper>

        {entities.length === 0 ? (
          <Paper radius="xl" p="xl" withBorder>
            <Stack align="center" gap="xl">
              <IconDatabase size={80} color="gray" opacity={0.3} />
              <Stack align="center" gap="xs">
                <Title order={3} c="dimmed">
                  No entities yet
                </Title>
                <Text c="dimmed" ta="center">
                  Add your first entity to start storing data
                </Text>
              </Stack>
              <Button
                leftSection={<IconPlus size={16} />}
                onClick={openCreateEntity}
                radius="xl"
                variant="gradient"
                gradient={{ from: 'blue', to: 'cyan' }}
              >
                Add First Entity
              </Button>
            </Stack>
          </Paper>
        ) : (
          <Paper withBorder radius="xl" p={0}>
            <ScrollArea>
              <Table highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>Entity ID</Table.Th>
                    {table.fields.map((field) => (
                      <Table.Th key={field.name}>
                        <Group gap="xs">
                          {field.name}
                          <Badge size="xs" variant="light">
                            {field.data_type}
                          </Badge>
                        </Group>
                      </Table.Th>
                    ))}
                    <Table.Th>Last Updated</Table.Th>
                    <Table.Th>Actions</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {entities.map((entity) => (
                    <EntityRow
                      key={entity.entity_id}
                      entity={entity}
                      table={table}
                      onEdit={openEditEntity}
                      onDelete={(id) => deleteEntityMutation.mutate(id)}
                      onUndelete={(id) => undeleteEntityMutation.mutate(id)}
                      onShowHistory={showHistory}
                    />
                  ))}
                </Table.Tbody>
              </Table>
            </ScrollArea>
          </Paper>
        )}
      </Stack>

      <Modal
        opened={entityModalOpened}
        onClose={closeEntityModal}
        title={
          <Group>
            <IconEdit size={20} />
            <Text fw={600}>{selectedEntity ? 'Edit Entity' : 'Create Entity'}</Text>
          </Group>
        }
        size="lg"
        radius="xl"
      >
        <form onSubmit={entityForm.onSubmit(handleSubmit)}>
          <Stack>
            {table.fields.map((field) => {
              const props = entityForm.getInputProps(field.name);

              switch (field.data_type) {
                case 'string':
                case 'reference':
                  return (
                    <TextInput
                      key={field.name}
                      label={field.name}
                      {...props}
                    />
                  );
                case 'int':
                  return (
                    <NumberInput
                      key={field.name}
                      label={field.name}
                      {...props}
                      step={1}
                    />
                  );
                case 'float':
                  return (
                    <NumberInput
                      key={field.name}
                      label={field.name}
                      {...props}
                      step={0.01}
                    />
                  );
                case 'bool':
                  return (
                    <Switch
                      key={field.name}
                      label={field.name}
                      {...props}
                      checked={props.value}
                    />
                  );
                case 'date':
                  return (
                    <DateTimePicker
                      key={field.name}
                      label={field.name}
                      {...props}
                    />
                  );
                case 'json':
                  return (
                    <JsonInput
                      key={field.name}
                      label={field.name}
                      placeholder='{"key": "value"}'
                      validationError="Invalid JSON"
                      formatOnBlur
                      autosize
                      minRows={4}
                      {...props}
                    />
                  );
                default:
                  return (
                    <TextInput
                      key={field.name}
                      label={field.name}
                      {...props}
                    />
                  );
              }
            })}

            <Group justify="flex-end" pt="md">
              <Button variant="subtle" onClick={closeEntityModal}>
                Cancel
              </Button>
              <Button
                type="submit"
                loading={createEntityMutation.isPending || updateEntityMutation.isPending}
                leftSection={selectedEntity ? <IconEdit size={16} /> : <IconPlus size={16} />}
              >
                {selectedEntity ? 'Update' : 'Create'}
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </>
  );
}

// Main app component
function MainApp() {
  const theme = useMantineTheme();
  const { user, logout } = useAuthStore();
  const [currentTable, setCurrentTable] = React.useState(null);

  const breadcrumbItems = currentTable
    ? [
        { title: 'Dashboard', href: '#' },
        { title: currentTable, href: '#' },
      ].map((item, index) => (
        <Anchor
          key={index}
          onClick={index === 0 ? () => setCurrentTable(null) : undefined}
          style={{ cursor: index === 0 ? 'pointer' : 'default' }}
        >
          {item.title}
        </Anchor>
      ))
    : [<Text key={0}>Dashboard</Text>];

  return (
    <AppShell
      header={{ height: 70 }}
      padding="md"
      styles={{
        main: {
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          minHeight: '100vh',
        },
      }}
    >
      <AppShell.Header>
        <Container size="xl" h="100%">
          <Group h="100%" justify="space-between">
            <Group>
              <IconDatabase size={32} color={theme.colors.blue[6]} />
              <Title order={2} fw={900} c="blue">
                Notably
              </Title>
            </Group>

            <Menu shadow="lg" radius="xl">
              <Menu.Target>
                <Group style={{ cursor: 'pointer' }}>
                  <Avatar color="blue" radius="xl">
                    <IconUser size={18} />
                  </Avatar>
                  <Stack gap={0}>
                    <Text fw={600}>{user}</Text>
                    <Text size="xs" c="dimmed">
                      Database User
                    </Text>
                  </Stack>
                </Group>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item leftSection={<IconUser size={14} />}>Profile</Menu.Item>
                <Menu.Item leftSection={<IconSettings size={14} />}>Settings</Menu.Item>
                <Menu.Divider />
                <Menu.Item leftSection={<IconLogout size={14} />} color="red" onClick={logout}>
                  Logout
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Container>
      </AppShell.Header>

      <AppShell.Main>
        <Container size="xl">
          <Paper radius="xl" p="xl" shadow="xl" style={{ backgroundColor: 'rgba(255, 255, 255, 0.95)' }}>
            <Stack gap="lg">
              <Breadcrumbs>{breadcrumbItems}</Breadcrumbs>

              {currentTable ? (
                <TableView tableId={currentTable} onBack={() => setCurrentTable(null)} />
              ) : (
                <TablesDashboard onSelectTable={setCurrentTable} />
              )}
            </Stack>
          </Paper>
        </Container>
      </AppShell.Main>
    </AppShell>
  );
}

// Root app component
function App() {
  const { isAuthenticated } = useAuthStore();

  return (
    <QueryClientProvider client={queryClient}>
      <MantineProvider
        theme={{
          primaryColor: 'blue',
          defaultRadius: 'md',
          components: {
            Button: {
              defaultProps: {
                radius: 'xl',
              },
            },
            Paper: {
              defaultProps: {
                radius: 'xl',
              },
            },
            Card: {
              defaultProps: {
                radius: 'xl',
              },
            },
          },
        }}
      >
        <ModalsProvider>
          <Notifications position="top-right" />
          {isAuthenticated() ? <MainApp /> : <AuthForm />}
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  );
}

export default App;
EOF

echo "🧹 Cleaning up unnecessary files..."
rm -f src/App.css src/index.css

echo "📦 Installing dependencies (this may take a moment)..."
npm install

echo ""
echo "🎉 Setup complete! Your Notably frontend is ready!"
echo ""
echo "🚀 To start the development server:"
echo "   cd notably-frontend"
echo "   npm run dev"
echo ""
echo "🔗 Then open: http://localhost:3000"
echo ""
echo "⚠️  Make sure your Notably API is running on http://localhost:8080"
echo ""
echo "✨ Features included:"
echo "   • Modern React + Vite + Mantine UI"
echo "   • Authentication with JWT"
echo "   • Dynamic table and entity management"
echo "   • Time travel queries"
echo "   • Beautiful responsive design"
echo "   • Real-time updates with React Query"
echo ""
echo "Happy coding! 🎨"
EOF

echo "✨ Setup script created! Here's how to use it:"
echo ""
echo "🎯 **Easy 3-step setup:**"
echo ""
echo "1. Make it executable:"
echo "   chmod +x setup-notably-frontend.sh"
echo ""
echo "2. Run the script:"
echo "   ./setup-notably-frontend.sh"
echo ""
echo "3. Start developing:"
echo "   cd notably-frontend"
echo "   npm run dev"
echo ""
echo "🔧 The script will:"
echo "   ✅ Check Node.js requirements"
echo "   ✅ Create the Vite project"
echo "   ✅ Install all Mantine dependencies"
echo "   ✅ Create all the beautiful UI files"
echo "   ✅ Set up API proxy to localhost:8080"
echo ""
echo "💡 **No manual copying needed!** Everything is automated."
