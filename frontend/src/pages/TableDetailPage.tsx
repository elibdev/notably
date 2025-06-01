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
  Table,
  ActionIcon,
  Menu,
  Loader,
  Alert,
  Modal,
  TextInput,
  Textarea,
  NumberInput,
  Switch,
  Flex,
  Box,
  Pagination,
  Breadcrumbs,
  Anchor,
  Select,
  Paper,
  Divider,
  Tooltip,
  ScrollArea,
} from '@mantine/core';
import {
  IconPlus,
  IconDots,
  IconEdit,
  IconTrash,
  IconHistory,
  IconArrowLeft,
  IconAlertCircle,
  IconRestore,
  IconCalendar,
  IconHash,
  IconDatabase,
} from '@tabler/icons-react';
import { useDisclosure } from '@mantine/hooks';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { Link, useParams, useNavigate } from 'react-router-dom';
import { DateInput } from '@mantine/dates';
import apiService from '../services/api';
import {
  TableResponse,
  EntityResponse,
  CreateEntityRequest,
  UpdateEntityRequest,
  EntityQueryParams,
  FieldResponse,
  ApiError,
} from '../types/api';

const ITEMS_PER_PAGE = 20;

const TableDetailPage: React.FC = () => {
  const { tableId } = useParams<{ tableId: string }>();
  const navigate = useNavigate();
  
  const [table, setTable] = useState<TableResponse | null>(null);
  const [entities, setEntities] = useState<EntityResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [entitiesLoading, setEntitiesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [includeDeleted, setIncludeDeleted] = useState(false);
  
  const [createModalOpened, { open: openCreateModal, close: closeCreateModal }] = useDisclosure(false);
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] = useDisclosure(false);
  const [selectedEntity, setSelectedEntity] = useState<EntityResponse | null>(null);
  const [deletingEntityId, setDeletingEntityId] = useState<string | null>(null);

  const createForm = useForm<CreateEntityRequest>({
    initialValues: {
      fields: {},
    },
  });

  const editForm = useForm<UpdateEntityRequest>({
    initialValues: {
      fields: {},
    },
  });

  const fetchTable = async () => {
    if (!tableId) return;
    
    try {
      setLoading(true);
      setError(null);
      const response = await apiService.getTable(tableId);
      setTable(response);
      
      // Initialize form with default values for each field
      const defaultFields: Record<string, any> = {};
      response.fields.forEach(field => {
        switch (field.data_type) {
          case 'string':
          case 'text':
            defaultFields[field.name] = '';
            break;
          case 'number':
          case 'integer':
          case 'float':
            defaultFields[field.name] = 0;
            break;
          case 'boolean':
            defaultFields[field.name] = false;
            break;
          case 'date':
          case 'datetime':
            defaultFields[field.name] = null;
            break;
          default:
            defaultFields[field.name] = '';
        }
      });
      createForm.setValues({ fields: defaultFields });
      
    } catch (err) {
      const apiError = err as ApiError;
      setError(apiError.message || 'Failed to fetch table details');
    } finally {
      setLoading(false);
    }
  };

  const fetchEntities = async (page: number = 1) => {
    if (!tableId) return;
    
    try {
      setEntitiesLoading(true);
      const params: EntityQueryParams = {
        limit: ITEMS_PER_PAGE,
        offset: (page - 1) * ITEMS_PER_PAGE,
        include_deleted: includeDeleted,
      };
      
      const response = await apiService.getEntities(tableId, params);
      setEntities(response.entities);
      setTotalPages(Math.ceil(response.entities.length / ITEMS_PER_PAGE));
      
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.message || 'Failed to fetch entities',
        color: 'red',
      });
    } finally {
      setEntitiesLoading(false);
    }
  };

  useEffect(() => {
    fetchTable();
  }, [tableId]);

  useEffect(() => {
    if (table) {
      fetchEntities(currentPage);
    }
  }, [table, currentPage, includeDeleted]);

  const handleCreateEntity = async (values: CreateEntityRequest) => {
    if (!tableId) return;
    
    try {
      await apiService.createEntity(tableId, values);
      
      notifications.show({
        title: 'Entity Created',
        message: 'New entity has been created successfully',
        color: 'green',
      });

      closeCreateModal();
      createForm.reset();
      fetchEntities(currentPage);
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Creation Failed',
        message: apiError.message || 'Failed to create entity',
        color: 'red',
      });
    }
  };

  const handleEditEntity = async (values: UpdateEntityRequest) => {
    if (!tableId || !selectedEntity) return;
    
    try {
      await apiService.updateEntity(tableId, selectedEntity.entity_id, values);
      
      notifications.show({
        title: 'Entity Updated',
        message: 'Entity has been updated successfully',
        color: 'green',
      });

      closeEditModal();
      setSelectedEntity(null);
      fetchEntities(currentPage);
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Update Failed',
        message: apiError.message || 'Failed to update entity',
        color: 'red',
      });
    }
  };

  const handleDeleteEntity = async (entityId: string) => {
    if (!tableId) return;
    
    try {
      setDeletingEntityId(entityId);
      await apiService.deleteEntity(tableId, entityId);
      
      notifications.show({
        title: 'Entity Deleted',
        message: 'Entity has been deleted successfully',
        color: 'blue',
      });

      fetchEntities(currentPage);
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Deletion Failed',
        message: apiError.message || 'Failed to delete entity',
        color: 'red',
      });
    } finally {
      setDeletingEntityId(null);
    }
  };

  const handleUndeleteEntity = async (entityId: string) => {
    if (!tableId) return;
    
    try {
      await apiService.undeleteEntity(tableId, entityId);
      
      notifications.show({
        title: 'Entity Restored',
        message: 'Entity has been restored successfully',
        color: 'green',
      });

      fetchEntities(currentPage);
    } catch (err) {
      const apiError = err as ApiError;
      notifications.show({
        title: 'Restore Failed',
        message: apiError.message || 'Failed to restore entity',
        color: 'red',
      });
    }
  };

  const openEditEntityModal = (entity: EntityResponse) => {
    setSelectedEntity(entity);
    editForm.setValues({ fields: entity.fields });
    openEditModal();
  };

  const renderFieldInput = (field: FieldResponse, form: any, prefix: string = 'fields') => {
    const fieldPath = `${prefix}.${field.name}`;
    
    switch (field.data_type) {
      case 'string':
        return (
          <TextInput
            key={field.name}
            label={field.name}
            {...form.getInputProps(fieldPath)}
          />
        );
      case 'text':
        return (
          <Textarea
            key={field.name}
            label={field.name}
            autosize
            minRows={2}
            {...form.getInputProps(fieldPath)}
          />
        );
      case 'number':
      case 'integer':
      case 'float':
        return (
          <NumberInput
            key={field.name}
            label={field.name}
            precision={field.data_type === 'float' ? 2 : 0}
            {...form.getInputProps(fieldPath)}
          />
        );
      case 'boolean':
        return (
          <Switch
            key={field.name}
            label={field.name}
            {...form.getInputProps(fieldPath, { type: 'checkbox' })}
          />
        );
      case 'date':
      case 'datetime':
        return (
          <DateInput
            key={field.name}
            label={field.name}
            valueFormat={field.data_type === 'date' ? 'YYYY-MM-DD' : 'YYYY-MM-DD HH:mm'}
            {...form.getInputProps(fieldPath)}
          />
        );
      default:
        return (
          <TextInput
            key={field.name}
            label={field.name}
            {...form.getInputProps(fieldPath)}
          />
        );
    }
  };

  const renderFieldValue = (field: FieldResponse, value: any) => {
    if (value === null || value === undefined) {
      return <Text c="dimmed" size="sm">-</Text>;
    }

    switch (field.data_type) {
      case 'boolean':
        return <Badge color={value ? 'green' : 'red'}>{value ? 'Yes' : 'No'}</Badge>;
      case 'date':
        return <Text size="sm">{new Date(value).toLocaleDateString()}</Text>;
      case 'datetime':
        return <Text size="sm">{new Date(value).toLocaleString()}</Text>;
      case 'text':
        return (
          <Tooltip label={value} multiline>
            <Text size="sm" truncate style={{ maxWidth: '200px' }}>
              {value}
            </Text>
          </Tooltip>
        );
      default:
        return <Text size="sm">{value.toString()}</Text>;
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
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

  if (!table) {
    return (
      <Container>
        <Alert
          icon={<IconAlertCircle size="1rem" />}
          color="red"
          title="Table Not Found"
        >
          The requested table could not be found.
        </Alert>
      </Container>
    );
  }

  const breadcrumbItems = [
    { title: 'Dashboard', href: '/dashboard' },
    { title: table.id, href: '' },
  ].map((item, index) => (
    <Anchor component={Link} to={item.href} key={index}>
      {item.title}
    </Anchor>
  ));

  return (
    <Container size="xl" py="md">
      <Stack gap="md">
        <Breadcrumbs mb="md">{breadcrumbItems}</Breadcrumbs>

        <Group justify="space-between">
          <div>
            <Group gap="md" mb="xs">
              <Title order={1}>{table.id}</Title>
              <Badge variant="light">{table.fields.length} fields</Badge>
            </Group>
            <Group gap="md">
              <Group gap="xs">
                <IconCalendar size={14} color="#868e96" />
                <Text size="sm" c="dimmed">
                  Created {formatDate(table.created_at)}
                </Text>
              </Group>
              <Group gap="xs">
                <IconDatabase size={14} color="#868e96" />
                <Text size="sm" c="dimmed">
                  Owner: {table.user_id}
                </Text>
              </Group>
            </Group>
          </div>
          <Group>
            <Button
              variant="subtle"
              leftSection={<IconArrowLeft size={16} />}
              onClick={() => navigate('/dashboard')}
            >
              Back
            </Button>
            <Button leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
              Add Entity
            </Button>
          </Group>
        </Group>

        {error && (
          <Alert
            icon={<IconAlertCircle size="1rem" />}
            color="red"
            variant="light"
            title="Error"
          >
            {error}
          </Alert>
        )}

        <Paper withBorder p="md" radius="md">
          <Group justify="space-between" mb="md">
            <Title order={3}>Entities</Title>
            <Group>
              <Switch
                label="Include deleted"
                checked={includeDeleted}
                onChange={(event) => setIncludeDeleted(event.currentTarget.checked)}
              />
            </Group>
          </Group>

          {entitiesLoading ? (
            <Flex justify="center" py="xl">
              <Loader />
            </Flex>
          ) : entities.length === 0 ? (
            <Box ta="center" py="xl">
              <IconDatabase size={48} stroke={1.5} color="#868e96" />
              <Title order={4} c="dimmed" mt="md">
                {includeDeleted ? 'No entities found' : 'No entities yet'}
              </Title>
              <Text c="dimmed" size="sm">
                {includeDeleted 
                  ? 'This table doesn\'t have any entities'
                  : 'Add your first entity to get started'
                }
              </Text>
              {!includeDeleted && (
                <Button mt="md" leftSection={<IconPlus size={16} />} onClick={openCreateModal}>
                  Add First Entity
                </Button>
              )}
            </Box>
          ) : (
            <>
              <ScrollArea>
                <Table striped highlightOnHover>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>ID</Table.Th>
                      {table.fields.map((field) => (
                        <Table.Th key={field.name}>{field.name}</Table.Th>
                      ))}
                      <Table.Th>Created</Table.Th>
                      <Table.Th>Status</Table.Th>
                      <Table.Th>Actions</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {entities.map((entity) => (
                      <Table.Tr key={entity.entity_id} opacity={entity.is_deleted ? 0.6 : 1}>
                        <Table.Td>
                          <Text size="sm" ff="monospace">
                            {entity.entity_id.slice(0, 8)}...
                          </Text>
                        </Table.Td>
                        {table.fields.map((field) => (
                          <Table.Td key={field.name}>
                            {renderFieldValue(field, entity.fields[field.name])}
                          </Table.Td>
                        ))}
                        <Table.Td>
                          <Text size="sm">{formatDate(entity.created_at)}</Text>
                        </Table.Td>
                        <Table.Td>
                          <Badge
                            color={entity.is_deleted ? 'red' : 'green'}
                            variant="light"
                          >
                            {entity.is_deleted ? 'Deleted' : 'Active'}
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          <Menu shadow="md" width={200} position="bottom-end">
                            <Menu.Target>
                              <ActionIcon variant="subtle" color="gray">
                                <IconDots size={16} />
                              </ActionIcon>
                            </Menu.Target>
                            <Menu.Dropdown>
                              {!entity.is_deleted ? (
                                <>
                                  <Menu.Item
                                    leftSection={<IconEdit size={14} />}
                                    onClick={() => openEditEntityModal(entity)}
                                  >
                                    Edit Entity
                                  </Menu.Item>
                                  <Menu.Item
                                    leftSection={<IconHistory size={14} />}
                                    component={Link}
                                    to={`/tables/${tableId}/entities/${entity.entity_id}/history`}
                                  >
                                    View History
                                  </Menu.Item>
                                  <Menu.Divider />
                                  <Menu.Item
                                    leftSection={<IconTrash size={14} />}
                                    color="red"
                                    onClick={() => handleDeleteEntity(entity.entity_id)}
                                    disabled={deletingEntityId === entity.entity_id}
                                  >
                                    {deletingEntityId === entity.entity_id ? 'Deleting...' : 'Delete'}
                                  </Menu.Item>
                                </>
                              ) : (
                                <Menu.Item
                                  leftSection={<IconRestore size={14} />}
                                  color="green"
                                  onClick={() => handleUndeleteEntity(entity.entity_id)}
                                >
                                  Restore Entity
                                </Menu.Item>
                              )}
                            </Menu.Dropdown>
                          </Menu>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              </ScrollArea>

              {totalPages > 1 && (
                <Group justify="center" mt="md">
                  <Pagination
                    value={currentPage}
                    onChange={setCurrentPage}
                    total={totalPages}
                  />
                </Group>
              )}
            </>
          )}
        </Paper>
      </Stack>

      {/* Create Entity Modal */}
      <Modal opened={createModalOpened} onClose={closeCreateModal} title="Create New Entity" size="md">
        <form onSubmit={createForm.onSubmit(handleCreateEntity)}>
          <Stack>
            {table.fields.map((field) => renderFieldInput(field, createForm))}
            
            <Group justify="flex-end" mt="md">
              <Button variant="subtle" onClick={closeCreateModal}>
                Cancel
              </Button>
              <Button type="submit">Create Entity</Button>
            </Group>
          </Stack>
        </form>
      </Modal>

      {/* Edit Entity Modal */}
      <Modal opened={editModalOpened} onClose={closeEditModal} title="Edit Entity" size="md">
        <form onSubmit={editForm.onSubmit(handleEditEntity)}>
          <Stack>
            {table.fields.map((field) => renderFieldInput(field, editForm))}
            
            <Group justify="flex-end" mt="md">
              <Button variant="subtle" onClick={closeEditModal}>
                Cancel
              </Button>
              <Button type="submit">Update Entity</Button>
            </Group>
          </Stack>
        </form>
      </Modal>
    </Container>
  );
};

export default TableDetailPage;