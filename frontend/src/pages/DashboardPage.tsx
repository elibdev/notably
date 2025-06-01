import React, { useState, useEffect } from 'react';
import {
  Container,
  Title,
  Button,
  Group,
  Stack,
  Card,
  Text,
  Badge,
  Grid,
  ActionIcon,
  Menu,
  Loader,
  Alert,
  Modal,
  TextInput,
  Select,
  Flex,
  Box,
} from '@mantine/core';
import {
  IconPlus,
  IconTable,
  IconCalendar,
  IconHash,
  IconDots,
  IconEdit,
  IconTrash,
  IconEye,
  IconAlertCircle,
} from '@tabler/icons-react';
import { useDisclosure } from '@mantine/hooks';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import apiService from '../services/api';
import { TableResponse, CreateTableRequest, FieldRequest, FieldDataType, ApiError } from '../types/api';

const FIELD_TYPES: { value: FieldDataType; label: string }[] = [
  { value: 'string', label: 'Text (String)' },
  { value: 'number', label: 'Number' },
  { value: 'integer', label: 'Integer' },
  { value: 'float', label: 'Float' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'date', label: 'Date' },
  { value: 'datetime', label: 'Date & Time' },
  { value: 'text', label: 'Long Text' },
];

const DashboardPage: React.FC = () => {
  const { user } = useAuth();
  const [tables, setTables] = useState<TableResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [deletingTableId, setDeletingTableId] = useState<string | null>(null);

  const createForm = useForm<CreateTableRequest & { initialFields: Array<{ name: string; type: FieldDataType }> }>({
    initialValues: {
      id: '',
      fields: [],
      initialFields: [{ name: 'name', type: 'string' }],
    },
    validate: {
      id: (value) => {
        if (!value.trim()) return 'Table ID is required';
        if (!/^[a-zA-Z0-9_-]+$/.test(value)) return 'Table ID can only contain letters, numbers, underscores, and hyphens';
        return null;
      },
      initialFields: {
        name: (value) => (!value?.trim() ? 'Field name is required' : null),
        type: (value) => (!value ? 'Field type is required' : null),
      },
    },
  });

  const fetchTables = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiService.getTables();
      setTables(response.tables);
    } catch (err) {
      const apiError = err as ApiError;
      setError(apiError.message || 'Failed to fetch tables');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTables();
  }, []);

  const handleCreateTable = async (values: CreateTableRequest & { initialFields: Array<{ name: string; type: FieldDataType }> }) => {
    try {
      const fields: FieldRequest[] = values.initialFields.map(field => ({
        name: field.name,
        data_type: field.type,
      }));

      const tableData: CreateTableRequest = {
        id: values.id,
        fields,
      };

      await apiService.createTable(tableData);
      
      notifications.show({
        title: 'Table Created',
        message: `Table "${values.id}" has been created successfully`,
        color: 'green',
      });

      closeCreateModal();
      createForm.reset();
      fetchTables();
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Creation Failed',
        message: apiError.message || 'Failed to create table',
        color: 'red',
      });
    }
  };

  const handleDeleteTable = async (tableId: string) => {
    try {
      setDeletingTableId(tableId);
      await apiService.deleteTable(tableId);
      
      notifications.show({
        title: 'Table Deleted',
        message: `Table "${tableId}" has been deleted`,
        color: 'blue',
      });

      fetchTables();
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Deletion Failed',
        message: apiError.message || 'Failed to delete table',
        color: 'red',
      });
    } finally {
      setDeletingTableId(null);
    }
  };

  const addField = () => {
    const currentFields = createForm.values.initialFields;
    createForm.setFieldValue('initialFields', [
      ...currentFields,
      { name: '', type: 'string' },
    ]);
  };

  const removeField = (index: number) => {
    const currentFields = createForm.values.initialFields;
    if (currentFields.length > 1) {
      createForm.setFieldValue('initialFields', currentFields.filter((_, i) => i !== index));
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  if (loading) {
    return (
      <Container>
        <Flex justify="center" align="center" h="50vh">
          <Loader size="lg" />
        </Flex>
      </Container>
    );
  }

  return (
    <Container size="xl" py="md">
      <Group justify="space-between" mb="xl">
        <div>
          <Title order={1}>Dashboard</Title>
          <Text c="dimmed" size="sm">
            Welcome back, {user?.user_id}
          </Text>
        </div>
        <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
          Create Table
        </Button>
      </Group>

      {error && (
        <Alert
          icon={<IconAlertCircle size="1rem" />}
          color="red"
          variant="light"
          mb="md"
          title="Error"
        >
          {error}
        </Alert>
      )}

      {tables.length === 0 ? (
        <Card withBorder p="xl" radius="md">
          <Stack align="center" gap="md">
            <IconTable size={48} stroke={1.5} color="#868e96" />
            <div style={{ textAlign: 'center' }}>
              <Title order={3} c="dimmed">No tables yet</Title>
              <Text c="dimmed" size="sm">
                Create your first table to start organizing your data
              </Text>
            </div>
            <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
              Create Your First Table
            </Button>
          </Stack>
        </Card>
      ) : (
        <Grid>
          {tables.map((table) => (
            <Grid.Col key={table.id} span={{ base: 12, sm: 6, lg: 4 }}>
              <Card withBorder shadow="sm" radius="md" h="100%">
                <Group justify="space-between" mb="xs">
                  <Text fw={500} size="lg" truncate>
                    {table.id}
                  </Text>
                  <Menu shadow="md" width={200} position="bottom-end">
                    <Menu.Target>
                      <ActionIcon variant="subtle" color="gray">
                        <IconDots size={16} />
                      </ActionIcon>
                    </Menu.Target>
                    <Menu.Dropdown>
                      <Menu.Item
                        leftSection={<IconEye size={14} />}
                        component={Link}
                        to={`/tables/${table.id}`}
                      >
                        View Table
                      </Menu.Item>
                      <Menu.Item
                        leftSection={<IconEdit size={14} />}
                        component={Link}
                        to={`/tables/${table.id}/edit`}
                      >
                        Edit Table
                      </Menu.Item>
                      <Menu.Divider />
                      <Menu.Item
                        leftSection={<IconTrash size={14} />}
                        color="red"
                        onClick={() => handleDeleteTable(table.id)}
                        disabled={deletingTableId === table.id}
                      >
                        {deletingTableId === table.id ? 'Deleting...' : 'Delete Table'}
                      </Menu.Item>
                    </Menu.Dropdown>
                  </Menu>
                </Group>

                <Stack gap="xs" flex={1}>
                  <Group gap="xs">
                    <IconHash size={14} color="#868e96" />
                    <Text size="sm" c="dimmed">
                      {table.fields.length} field{table.fields.length !== 1 ? 's' : ''}
                    </Text>
                  </Group>

                  <Group gap="xs">
                    <IconCalendar size={14} color="#868e96" />
                    <Text size="sm" c="dimmed">
                      Created {formatDate(table.created_at)}
                    </Text>
                  </Group>

                  <Box mt="xs">
                    <Text size="xs" c="dimmed" mb={4}>Fields:</Text>
                    <Group gap={4}>
                      {table.fields.slice(0, 3).map((field) => (
                        <Badge key={field.name} size="xs" variant="light">
                          {field.name}
                        </Badge>
                      ))}
                      {table.fields.length > 3 && (
                        <Badge size="xs" variant="outline" c="dimmed">
                          +{table.fields.length - 3} more
                        </Badge>
                      )}
                    </Group>
                  </Box>
                </Stack>

                <Button
                  component={Link}
                  to={`/tables/${table.id}`}
                  variant="light"
                  fullWidth
                  mt="md"
                  leftSection={<IconEye size={14} />}
                >
                  View Table
                </Button>
              </Card>
            </Grid.Col>
          ))}
        </Grid>
      )}

      <Modal opened={createModalOpened} onClose={closeCreateModal} title="Create New Table" size="md">
        <form onSubmit={createForm.onSubmit(handleCreateTable)}>
          <Stack>
            <TextInput
              label="Table ID"
              placeholder="Enter table identifier"
              required
              {...createForm.getInputProps('id')}
            />

            <div>
              <Text size="sm" fw={500} mb="xs">
                Initial Fields
              </Text>
              {createForm.values.initialFields.map((field, index) => (
                <Group key={index} mb="xs" align="end">
                  <TextInput
                    placeholder="Field name"
                    flex={1}
                    {...createForm.getInputProps(`initialFields.${index}.name`)}
                  />
                  <Select
                    placeholder="Type"
                    data={FIELD_TYPES}
                    style={{ flexBasis: '140px' }}
                    {...createForm.getInputProps(`initialFields.${index}.type`)}
                  />
                  <ActionIcon
                    color="red"
                    variant="subtle"
                    onClick={() => removeField(index)}
                    disabled={createForm.values.initialFields.length === 1}
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Group>
              ))}
              <Button
                variant="subtle"
                leftSection={<IconPlus size={14} />}
                onClick={addField}
                size="xs"
              >
                Add Field
              </Button>
            </div>

            <Group justify="flex-end" mt="md">
              <Button variant="subtle" onClick={closeCreateModal}>
                Cancel
              </Button>
              <Button type="submit">Create Table</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Container>
  );
};

export default DashboardPage;