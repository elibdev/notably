import React, { useState, useEffect, useMemo, useCallback } from "react";
import {
  ApiClient,
  type TableInfo,
  type RowData,
  type RowEvent,
  type ColumnDefinition,
} from "./api";
import { BrowserRouter, Routes, Route, useParams, useNavigate, Navigate } from "react-router-dom";

// Mantine UI components
import {
  AppShell,
  Button,
  Flex,
  Text,
  Title,
  Container,
  Paper,
  Group,
  Anchor,
  Center,
  Loader,
  TextInput,
  ActionIcon,
  Table,
  Badge,
  Modal,
  Grid,
  Select,
  Accordion,
  ScrollArea,
  Box,
  PasswordInput,
  Card,
  Stack,
  LoadingOverlay,
  Tabs,
  Drawer,
  Tooltip,
  Code,
  JsonInput,
  Divider,
} from "@mantine/core";

import { DateTimePicker } from "@mantine/dates";

// Mantine hooks
import { useDisclosure } from "@mantine/hooks";
import { useForm } from "@mantine/form";

// Notifications
import { notifications } from "@mantine/notifications";

// Icons
import {
  IconDatabase,
  IconTable,
  IconLogout,
  IconUserPlus,
  IconPlus,
  IconTrash,
  IconEdit,
  IconHistory,
  IconClock,
  IconChevronLeft,
  IconCheck,
  IconX,
} from "@tabler/icons-react";

// Define type for error with message
interface ErrorWithMessage {
  message: string;
}

// Type guard for ErrorWithMessage
function isErrorWithMessage(error: unknown): error is ErrorWithMessage {
  return (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof (error as Record<string, unknown>).message === "string"
  );
}

// Function to get error message from unknown error
function getErrorMessage(error: unknown): string {
  if (isErrorWithMessage(error)) {
    return error.message;
  }
  return String(error);
}

export function App() {
  const [apiKey, setApiKey] = useState<string>(localStorage.getItem("apiKey") || "");
  const [client, setClient] = useState<ApiClient | null>(null);

  useEffect(() => {
    if (apiKey) {
      setClient(new ApiClient(apiKey));
    } else {
      setClient(null);
    }
  }, [apiKey]);

  const handleLogout = () => {
    setApiKey("");
    localStorage.removeItem("apiKey");
    notifications.show({
      title: "Logged out",
      message: "You have been successfully logged out",
      color: "blue",
    });
  };

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={
            client ? (
              <Navigate to="/tables" replace />
            ) : (
              <AuthForm
                title="Login to Notably"
                onSubmit={async ({ username, password }) => {
                  try {
                    const res = await ApiClient.login(username, password);
                    setApiKey(res.apiKey);
                    localStorage.setItem("apiKey", res.apiKey);
                    notifications.show({
                      title: "Welcome back!",
                      message: `Successfully logged in as ${username}`,
                      color: "green",
                    });
                  } catch (error: unknown) {
                    notifications.show({
                      title: "Login failed",
                      message: getErrorMessage(error),
                      color: "red",
                    });
                    throw error;
                  }
                }}
                onSwitch={() => "/register"}
              />
            )
          }
        />
        <Route
          path="/register"
          element={
            client ? (
              <Navigate to="/tables" replace />
            ) : (
              <AuthForm
                title="Create an Account"
                includeEmail
                onSubmit={async ({ username, email, password }) => {
                  try {
                    const res = await ApiClient.register(username, email!, password);
                    setApiKey(res.apiKey);
                    localStorage.setItem("apiKey", res.apiKey);
                    notifications.show({
                      title: "Account created",
                      message: "Your account has been successfully created",
                      color: "green",
                    });
                  } catch (error: unknown) {
                    notifications.show({
                      title: "Registration failed",
                      message: getErrorMessage(error),
                      color: "red",
                    });
                    throw error;
                  }
                }}
                onSwitch={() => "/login"}
              />
            )
          }
        />
        <Route
          path="/tables"
          element={
            !client ? (
              <Navigate to="/login" replace />
            ) : (
              <MainApp client={client} onLogout={handleLogout} />
            )
          }
        />
        <Route
          path="/tables/:tableName"
          element={
            !client ? (
              <Navigate to="/login" replace />
            ) : (
              <TableDetail client={client} onLogout={handleLogout} />
            )
          }
        />
        <Route path="/" element={<Navigate to={client ? "/tables" : "/login"} replace />} />
      </Routes>
    </BrowserRouter>
  );
}

interface AuthFormProps {
  title: string;
  includeEmail?: boolean;
  onSubmit: (fields: { username: string; email?: string; password: string }) => Promise<void>;
  onSwitch: () => string;
}

function AuthForm({ title, includeEmail, onSubmit, onSwitch }: AuthFormProps) {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await onSubmit({ username, email, password });
      navigate("/tables");
    } catch {
      // Error is handled by notifications in the parent component
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Card shadow="md" radius="md" p="xl" withBorder>
        <Card.Section bg="blue.6" p="md">
          <Title order={2} c="white">
            {title}
          </Title>
        </Card.Section>

        <form onSubmit={handleSubmit}>
          <Stack spacing="md" mt="md" pos="relative">
            <LoadingOverlay
              visible={loading}
              zIndex={1000}
              overlayProps={{ radius: "sm", blur: 2 }}
            />

            <TextInput
              label="Username"
              placeholder="johndoe"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              icon={<IconUserPlus size={16} />}
            />

            {includeEmail && (
              <TextInput
                label="Email"
                placeholder="example@email.com"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            )}

            <PasswordInput
              label="Password"
              placeholder="Your secure password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />

            <Button type="submit" fullWidth color="blue" mt="md">
              {includeEmail ? "Create Account" : "Login"}
            </Button>

            <Group position="center" mt="md">
              <Text size="sm">
                {includeEmail ? "Already have an account?" : "Don't have an account?"}
              </Text>
              <Anchor
                onClick={() => navigate(onSwitch())}
                size="sm"
                color="blue"
                style={{ cursor: "pointer" }}
              >
                {includeEmail ? "Login" : "Register"}
              </Anchor>
            </Group>
          </Stack>
        </form>
      </Card>
    </Container>
  );
}

interface MainAppProps {
  client: ApiClient;
  onLogout: () => void;
}

function MainApp({ client, onLogout }: MainAppProps) {
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [createModalOpen, { open: openCreateModal, close: closeCreateModal }] =
    useDisclosure(false);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  // Form for creating a new table with columns
  const form = useForm({
    initialValues: {
      name: "",
      columns: [{ name: "", dataType: "string" }],
    },
    validate: {
      name: (value) => (!value ? "Table name is required" : null),
      columns: {
        name: (value) => (!value ? "Column name is required" : null),
        dataType: (value) => (!value ? "Data type is required" : null),
      },
    },
  });

  // Memoize loadTables to prevent recreation on every render
  const loadTables = useCallback(async () => {
    setLoading(true);
    try {
      const res = await client.listTables();
      setTables(res.tables);
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    loadTables();
  }, [loadTables]);

  const handleCreateTable = async (values: { name: string; columns: ColumnDefinition[] }) => {
    setLoading(true);
    try {
      await client.createTable(values.name, values.columns);
      notifications.show({
        title: "Success",
        message: `Table "${values.name}" created successfully`,
        color: "green",
      });
      form.reset();
      closeCreateModal();
      loadTables();
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error",
          message: error.message,
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppShell header={{ height: 60 }} padding="md">
      <Flex justify="space-between" align="center" h="100%">
        <Group>
          <IconDatabase size={24} />
          <Title order={3}>Notably</Title>
        </Group>
        <Button variant="subtle" color="red" onClick={onLogout} lefticon={<IconLogout size={16} />}>
          Logout
        </Button>
      </Flex>

      <Container size="lg" py="md">
        <Paper p="md" shadow="sm" radius="md" withborder="true">
          <Flex justify="space-between" align="center" mb="md">
            <Title order={4}>Your Tables</Title>
            <Button lefticon={<IconPlus size={16} />} onClick={openCreateModal}>
              Create Table
            </Button>
          </Flex>

          {tables.length === 0 ? (
            <Center py="xl">
              <Stack align="center" spacing="sm">
                <IconTable size={48} opacity={0.3} />
                <Text c="dimmed">No tables yet. Create your first table to get started.</Text>
              </Stack>
            </Center>
          ) : (
            <Table striped highlightOnHover withborder="true">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Created</th>
                  <th>Columns</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {tables.map((table) => (
                  <tr key={table.name}>
                    <td>
                      <Text fw={500}>{table.name}</Text>
                    </td>
                    <td>{new Date(table.createdAt).toLocaleString()}</td>
                    <td>
                      <Group spacing="xs">
                        {table.columns?.map((col) => (
                          <Badge key={col.name}>
                            {col.name}: {col.dataType}
                          </Badge>
                        ))}
                        {!table.columns?.length && (
                          <Text size="sm" c="dimmed">
                            No schema defined
                          </Text>
                        )}
                      </Group>
                    </td>
                    <td>
                      <Button
                        variant="light"
                        size="xs"
                        onClick={() => navigate(`/tables/${encodeURIComponent(table.name)}`)}
                      >
                        Open
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </Paper>
      </Container>

      {/* Create Table Modal */}
      <Modal opened={createModalOpen} onClose={closeCreateModal} title="Create New Table" size="lg">
        <form onSubmit={form.onSubmit(handleCreateTable)}>
          <TextInput
            label="Table Name"
            placeholder="Enter table name"
            required
            {...form.getInputProps("name")}
            mb="md"
          />

          <Title order={5} mb="xs">
            Column Definitions
          </Title>

          <Box mb="md">
            {form.values.columns.map((_, index) => (
              <Grid key={index} mb="xs" align="flex-end">
                <Grid.Col span={5}>
                  <TextInput
                    label={index === 0 ? "Column Name" : ""}
                    placeholder="column_name"
                    {...form.getInputProps(`columns.${index}.name`)}
                  />
                </Grid.Col>
                <Grid.Col span={5}>
                  <Select
                    label={index === 0 ? "Data Type" : ""}
                    placeholder="Select data type"
                    data={[
                      { value: "string", label: "String" },
                      { value: "number", label: "Number" },
                      { value: "boolean", label: "Boolean" },
                      { value: "datetime", label: "DateTime" },
                      { value: "object", label: "Object/JSON" },
                      { value: "array", label: "Array" },
                    ]}
                    {...form.getInputProps(`columns.${index}.dataType`)}
                  />
                </Grid.Col>
                <Grid.Col span={2}>
                  <Group>
                    {index === form.values.columns.length - 1 && (
                      <ActionIcon
                        color="blue"
                        onClick={() =>
                          form.insertListItem("columns", { name: "", dataType: "string" })
                        }
                      >
                        <IconPlus size={16} />
                      </ActionIcon>
                    )}
                    {form.values.columns.length > 1 && (
                      <ActionIcon color="red" onClick={() => form.removeListItem("columns", index)}>
                        <IconTrash size={16} />
                      </ActionIcon>
                    )}
                  </Group>
                </Grid.Col>
              </Grid>
            ))}
          </Box>

          <Group position="right" mt="md">
            <Button variant="subtle" onClick={closeCreateModal}>
              Cancel
            </Button>
            <Button type="submit" loading={loading}>
              Create Table
            </Button>
          </Group>
        </form>
      </Modal>
    </AppShell>
  );
}

interface TableViewProps {
  table: string;
  client: ApiClient;
  onBack: () => void;
  tableInfo?: TableInfo;
}

// Add a new TableDetail component that uses URL parameters
function TableDetail({ client }: MainAppProps) {
  const { tableName } = useParams<{ tableName: string }>();
  const navigate = useNavigate();
  const [tableInfo, setTableInfo] = useState<TableInfo | undefined>(undefined);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadTableInfo() {
      setLoading(true);
      try {
        const res = await client.listTables();
        const foundTable = res.tables.find((t) => t.name === tableName);
        setTableInfo(foundTable);
      } catch (error) {
        notifications.show({
          title: "Error",
          message: getErrorMessage(error),
          color: "red",
        });
        navigate("/tables");
      } finally {
        setLoading(false);
      }
    }

    if (tableName) {
      loadTableInfo();
    }
  }, [tableName, client, navigate]);

  if (loading) {
    return (
      <Center style={{ height: "100vh" }}>
        <Loader size="xl" />
      </Center>
    );
  }

  if (!tableName || !tableInfo) {
    return <Navigate to="/tables" replace />;
  }

  return (
    <TableView
      table={tableName}
      client={client}
      onBack={() => navigate("/tables")}
      tableInfo={tableInfo}
    />
  );
}

function TableView({ table, client, onBack, tableInfo }: TableViewProps) {
  const [rows, setRows] = useState<RowData[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<string | null>("rows");
  const [newRowDrawerOpen, { open: openNewRowDrawer, close: closeNewRowDrawer }] =
    useDisclosure(false);
  const [editRowDrawerOpen, { open: openEditRowDrawer, close: closeEditRowDrawer }] =
    useDisclosure(false);
  const [currentEditingRow, setCurrentEditingRow] = useState<string>("");
  const [newRowValues, setNewRowValues] = useState<Record<string, unknown>>({});
  const [newRowId, setNewRowId] = useState("");

  const [snapshotAt, setSnapshotAt] = useState<Date | null>(null);
  const [historyRange, setHistoryRange] = useState({
    start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000),
    end: new Date(),
  });
  const [historyEvents, setHistoryEvents] = useState<RowEvent[]>([]);

  // Memoize columns to prevent unnecessary re-renders
  const columns = useMemo(() => tableInfo?.columns || [], [tableInfo?.columns]);

  // Create a form for new row with default values based on column definitions
  useEffect(() => {
    const defaultValues: Record<string, unknown> = {};
    columns.forEach((col) => {
      switch (col.dataType) {
        case "string":
          defaultValues[col.name] = "";
          break;
        case "number":
          defaultValues[col.name] = 0;
          break;
        case "boolean":
          defaultValues[col.name] = false;
          break;
        case "datetime":
          defaultValues[col.name] = new Date().toISOString();
          break;
        case "object":
          defaultValues[col.name] = {};
          break;
        case "array":
          defaultValues[col.name] = [];
          break;
        default:
          defaultValues[col.name] = "";
      }
    });
    setNewRowValues(defaultValues);
  }, [columns]);

  // Memoize loadRows to prevent recreation on every render
  const loadRows = useCallback(async () => {
    setLoading(true);
    try {
      const res = await client.listRows(table);
      setRows(res.rows);
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error loading rows",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  }, [client, table]);

  useEffect(() => {
    loadRows();
  }, [table, loadRows]);

  const applySnapshot = async () => {
    setLoading(true);
    try {
      const atIso = snapshotAt ? snapshotAt.toISOString() : undefined;
      const res = await client.snapshot(table, atIso);
      setRows(res.rows);
      notifications.show({
        title: "Snapshot loaded",
        message: `Loaded table snapshot from ${snapshotAt?.toLocaleString() || "latest"}`,
        color: "blue",
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error loading snapshot",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const loadHistory = async () => {
    setLoading(true);
    try {
      const startIso = historyRange.start.toISOString();
      const endIso = historyRange.end.toISOString();
      const e = await client.history(table, startIso, endIso);
      setHistoryEvents(e.events);
      notifications.show({
        title: "History loaded",
        message: `Loaded ${e.events.length} history events`,
        color: "blue",
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error loading history",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const addRow = async () => {
    setLoading(true);
    try {
      // Use provided ID or let the backend generate one
      const rowId = newRowId.trim()
        ? newRowId
        : `row_${Math.random().toString(36).substring(2, 11)}`;
      await client.createRow(table, rowId, newRowValues);
      closeNewRowDrawer();
      setNewRowId("");
      loadRows();
      notifications.show({
        title: "Row added",
        message: `Added row with ID ${rowId}`,
        color: "green",
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error adding row",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const updateRow = async (id: string, values: Record<string, unknown>) => {
    setLoading(true);
    try {
      await client.updateRow(table, id, values);
      closeEditRowDrawer();
      loadRows();
      notifications.show({
        title: "Row updated",
        message: `Updated row ${id}`,
        color: "green",
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error updating row",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const deleteRow = async (id: string) => {
    if (!window.confirm(`Delete row ${id}?`)) return;
    setLoading(true);
    try {
      await client.deleteRow(table, id);
      loadRows();
      notifications.show({
        title: "Row deleted",
        message: `Deleted row ${id}`,
        color: "blue",
      });
    } catch (error: unknown) {
      if (error instanceof Error) {
        notifications.show({
          title: "Error deleting row",
          message: getErrorMessage(error),
          color: "red",
        });
      }
    } finally {
      setLoading(false);
    }
  };

  const startEditRow = (id: string) => {
    const row = rows.find((r) => r.id === id);
    if (row) {
      setNewRowValues(row.values);
      setCurrentEditingRow(id);
      openEditRowDrawer();
    }
  };

  const renderInputForType = (
    columnName: string,
    dataType: string,
    value: unknown,
    onChange: (value: unknown) => void,
  ) => {
    switch (dataType) {
      case "string":
        return (
          <TextInput
            value={value || ""}
            onChange={(e) => onChange(e.target.value)}
            placeholder={`Enter ${columnName}`}
          />
        );
      case "number":
        return (
          <TextInput
            type="number"
            value={value?.toString() || "0"}
            onChange={(e) => onChange(parseFloat(e.target.value))}
            placeholder={`Enter ${columnName}`}
          />
        );
      case "boolean":
        return (
          <Select
            value={value?.toString() || "false"}
            onChange={(val) => onChange(val === "true")}
            data={[
              { value: "true", label: "True" },
              { value: "false", label: "False" },
            ]}
          />
        );
      case "datetime":
        return (
          <DateTimePicker
            value={value ? new Date(value) : new Date()}
            onChange={(date) => onChange(date?.toISOString())}
          />
        );
      case "object":
      case "array":
      case "json":
        return (
          <JsonInput
            value={JSON.stringify(value || (dataType === "array" ? [] : {}), null, 2)}
            onChange={(val) => {
              try {
                onChange(JSON.parse(val));
              } catch {
                // Invalid JSON, don't update
              }
            }}
            formatOnBlur
            autosize
            minRows={4}
          />
        );
      default:
        return (
          <TextInput value={value?.toString() || ""} onChange={(e) => onChange(e.target.value)} />
        );
    }
  };

  return (
    <AppShell header={{ height: 60 }} padding="md">
      <Flex justify="space-between" align="center" h="100%">
        <Group>
          <Button variant="subtle" lefticon={<IconChevronLeft size={16} />} onClick={onBack}>
            Back to Tables
          </Button>
          <Title order={3}>{table}</Title>
        </Group>
        <Group>
          <Badge size="lg">{rows.length} rows</Badge>
          <Button onClick={openNewRowDrawer} lefticon={<IconPlus size={16} />}>
            Add Row
          </Button>
        </Group>
      </Flex>

      <Container size="lg" py="md">
        <Tabs value={activeTab} onChange={setActiveTab}>
          <Tabs.List>
            <Tabs.Tab value="rows" icon={<IconTable size={16} />}>
              Rows
            </Tabs.Tab>
            <Tabs.Tab value="snapshot" icon={<IconClock size={16} />}>
              Snapshot
            </Tabs.Tab>
            <Tabs.Tab value="history" icon={<IconHistory size={16} />}>
              History
            </Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="rows" pt="md">
            <Paper p="md" withborder="true">
              {rows.length === 0 ? (
                <Center py="xl">
                  <Stack align="center" spacing="sm">
                    <IconTable size={48} opacity={0.3} />
                    <Text c="dimmed">No rows yet. Add your first row to get started.</Text>
                    <Button mt="md" onClick={openNewRowDrawer} leftIcon={<IconPlus size={16} />}>
                      Add Row
                    </Button>
                  </Stack>
                </Center>
              ) : (
                <ScrollArea>
                  <Table striped highlightOnHover withborder="true">
                    <thead>
                      <tr>
                        <th>ID</th>
                        <th>Timestamp</th>
                        {columns.map((col) => (
                          <th key={col.name}>{col.name}</th>
                        ))}
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((row) => (
                        <tr key={row.id}>
                          <td>
                            <Text fw={500}>{row.id}</Text>
                          </td>
                          <td>
                            <Text size="sm">{new Date(row.timestamp).toLocaleString()}</Text>
                          </td>
                          {columns.map((col) => (
                            <td key={col.name}>
                              {col.dataType === "object" || col.dataType === "array" ? (
                                <Code block>{JSON.stringify(row.values[col.name], null, 2)}</Code>
                              ) : col.dataType === "boolean" ? (
                                row.values[col.name] ? (
                                  <Badge color="green">
                                    <IconCheck size={14} />
                                    True
                                  </Badge>
                                ) : (
                                  <Badge color="red">
                                    <IconX size={14} />
                                    False
                                  </Badge>
                                )
                              ) : (
                                <Text>{String(row.values[col.name] || "")}</Text>
                              )}
                            </td>
                          ))}
                          <td>
                            <Group spacing="xs">
                              <Tooltip label="Edit">
                                <ActionIcon color="blue" onClick={() => startEditRow(row.id)}>
                                  <IconEdit size={16} />
                                </ActionIcon>
                              </Tooltip>
                              <Tooltip label="Delete">
                                <ActionIcon color="red" onClick={() => deleteRow(row.id)}>
                                  <IconTrash size={16} />
                                </ActionIcon>
                              </Tooltip>
                            </Group>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </Table>
                </ScrollArea>
              )}
            </Paper>
          </Tabs.Panel>

          <Tabs.Panel value="snapshot" pt="md">
            <Paper p="md" withBorder>
              <Grid>
                <Grid.Col span={9}>
                  <DateTimePicker
                    label="Snapshot Time"
                    placeholder="Select time"
                    value={snapshotAt}
                    onChange={setSnapshotAt}
                    clearable
                  />
                </Grid.Col>
                <Grid.Col span={3}>
                  <Button mt={24} onClick={applySnapshot} loading={loading} fullWidth>
                    Load Snapshot
                  </Button>
                </Grid.Col>
              </Grid>

              <Divider my="md" label="Snapshot Results" labelPosition="center" />

              {rows.length > 0 ? (
                <Table striped highlightOnHover withborder="true">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Timestamp</th>
                      {columns.map((col) => (
                        <th key={col.name}>{col.name}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.id}>
                        <td>
                          <Text fw={500}>{row.id}</Text>
                        </td>
                        <td>
                          <Text size="sm">{new Date(row.timestamp).toLocaleString()}</Text>
                        </td>
                        {columns.map((col) => (
                          <td key={col.name}>
                            {col.dataType === "object" || col.dataType === "array" ? (
                              <Code block>{JSON.stringify(row.values[col.name], null, 2)}</Code>
                            ) : (
                              <Text>{String(row.values[col.name] || "")}</Text>
                            )}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </Table>
              ) : (
                <Center py="xl">
                  <Text c="dimmed">No data in this snapshot</Text>
                </Center>
              )}
            </Paper>
          </Tabs.Panel>

          <Tabs.Panel value="history" pt="md">
            <Paper p="md" withborder="true">
              <Grid>
                <Grid.Col span={4}>
                  <DateTimePicker
                    label="Start Time"
                    placeholder="Select start time"
                    value={historyRange.start}
                    onChange={(date) =>
                      date && setHistoryRange((prev) => ({ ...prev, start: date }))
                    }
                  />
                </Grid.Col>
                <Grid.Col span={4}>
                  <DateTimePicker
                    label="End Time"
                    placeholder="Select end time"
                    value={historyRange.end}
                    onChange={(date) => date && setHistoryRange((prev) => ({ ...prev, end: date }))}
                  />
                </Grid.Col>
                <Grid.Col span={4}>
                  <Button mt={24} onClick={loadHistory} loading={loading} fullWidth>
                    Load History
                  </Button>
                </Grid.Col>
              </Grid>

              <Divider my="md" label="History Events" labelPosition="center" />

              {historyEvents.length > 0 ? (
                <Accordion>
                  {historyEvents.map((event) => (
                    <Accordion.Item
                      key={`${event.id}-${event.timestamp}`}
                      value={`${event.id}-${event.timestamp}`}
                    >
                      <Accordion.Control>
                        <Group>
                          <Text fw={500}>{event.id}</Text>
                          <Badge>{new Date(event.timestamp).toLocaleString()}</Badge>
                          <Text c="dimmed" size="sm">
                            {event.values ? "Update" : "Delete"}
                          </Text>
                        </Group>
                      </Accordion.Control>
                      <Accordion.Panel>
                        {event.values ? (
                          <JsonInput
                            value={JSON.stringify(event.values, null, 2)}
                            readOnly
                            formatOnBlur
                            autosize
                            minRows={4}
                          />
                        ) : (
                          <Text c="dimmed">Row was deleted at this point</Text>
                        )}
                      </Accordion.Panel>
                    </Accordion.Item>
                  ))}
                </Accordion>
              ) : (
                <Center py="xl">
                  <Text c="dimmed">No history events in the selected time range</Text>
                </Center>
              )}
            </Paper>
          </Tabs.Panel>
        </Tabs>
      </Container>

      {/* New Row Drawer */}
      <Drawer
        opened={newRowDrawerOpen}
        onClose={closeNewRowDrawer}
        title="Add New Row"
        position="right"
        size="lg"
      >
        <Paper p="md" withborder="true">
          <TextInput
            label="Row ID (optional)"
            description="Unique identifier for this row - will be auto-generated if left empty"
            placeholder="Enter row ID or leave empty for auto-generation"
            value={newRowId}
            onChange={(e) => setNewRowId(e.target.value)}
            mb="md"
          />

          <Title order={5} mb="md">
            Row Values
          </Title>
          {columns.map((col) => (
            <Box key={col.name} mb="md">
              <Text fw={500} mb={5}>
                {col.name} <Badge>{col.dataType}</Badge>
              </Text>
              {renderInputForType(col.name, col.dataType, newRowValues[col.name], (val) =>
                setNewRowValues((prev) => ({ ...prev, [col.name]: val })),
              )}
            </Box>
          ))}

          <Group position="right" mt="xl">
            <Button variant="subtle" onClick={closeNewRowDrawer}>
              Cancel
            </Button>
            <Button onClick={addRow} disabled={!newRowId.trim()} loading={loading}>
              Create Row
            </Button>
          </Group>
        </Paper>
      </Drawer>

      {/* Edit Row Drawer */}
      <Drawer
        opened={editRowDrawerOpen}
        onClose={closeEditRowDrawer}
        title={`Edit Row: ${currentEditingRow}`}
        position="right"
        size="lg"
      >
        <Paper p="md" withborder="true">
          <Title order={5} mb="md">
            Row Values
          </Title>
          {columns.map((col) => (
            <Box key={col.name} mb="md">
              <Text fw={500} mb={5}>
                {col.name} <Badge>{col.dataType}</Badge>
              </Text>
              {renderInputForType(col.name, col.dataType, newRowValues[col.name], (val) =>
                setNewRowValues((prev) => ({ ...prev, [col.name]: val })),
              )}
            </Box>
          ))}

          <Group position="right" mt="xl">
            <Button variant="subtle" onClick={closeEditRowDrawer}>
              Cancel
            </Button>
            <Button onClick={() => updateRow(currentEditingRow, newRowValues)} loading={loading}>
              Update Row
            </Button>
          </Group>
        </Paper>
      </Drawer>
    </AppShell>
  );
}

export default App;
