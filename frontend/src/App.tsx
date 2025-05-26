import React, { useState, useEffect } from "react";
import { ApiClient, type TableInfo } from "./api";
import { BrowserRouter, Routes, Route, useNavigate, Navigate } from "react-router-dom";

import {
  AppShell,
  Button,
  Flex,
  Text,
  Title,
  Container,
  Paper,
  Group,
  Center,
  TextInput,
  Table,
  Badge,
  Card,
  Stack,
  LoadingOverlay,
  Tabs,
  PasswordInput,
} from "@mantine/core";

import { notifications } from "@mantine/notifications";
import { IconDatabase, IconTable, IconLogout, IconUserPlus, IconPlus } from "@tabler/icons-react";

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  return "An unknown error occurred";
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
          path="/auth"
          element={
            client ? (
              <Navigate to="/tables" replace />
            ) : (
              <AuthComponent
                onLogin={async (username, password) => {
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
                onRegister={async (username, email, password) => {
                  try {
                    const res = await ApiClient.register(username, email, password);
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
              />
            )
          }
        />
        <Route
          path="/tables"
          element={
            !client ? (
              <Navigate to="/auth" replace />
            ) : (
              <MainApp client={client} onLogout={handleLogout} />
            )
          }
        />
        <Route path="/" element={<Navigate to={client ? "/tables" : "/auth"} replace />} />
        <Route path="/login" element={<Navigate to="/auth" replace />} />
        <Route path="/register" element={<Navigate to="/auth" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

interface AuthComponentProps {
  onLogin: (username: string, password: string) => Promise<void>;
  onRegister: (username: string, email: string, password: string) => Promise<void>;
}

function AuthComponent({ onLogin, onRegister }: AuthComponentProps) {
  const [activeTab, setActiveTab] = useState("login");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const isRegisterTab = activeTab === "register";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    let validationError = "";
    if (!username.trim()) {
      validationError = "Username is required.";
    } else if (isRegisterTab && !email.trim()) {
      validationError = "Email is required for registration.";
    } else if (!password.trim()) {
      validationError = "Password is required.";
    }

    if (validationError) {
      notifications.show({ // From '@mantine/notifications'
        title: "Validation Error",
        message: validationError,
        color: "red",
      });
      return; // Stop submission
    }

    setLoading(true);
    try {
      if (isRegisterTab) {
        await onRegister(username, email, password);
      } else {
        await onLogin(username, password);
      }
      navigate("/tables");
    } catch {
      // Error is handled by notifications in the parent component
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size="xs" py="xl">
      <Card shadow="md" radius="md" p="xl" withBorder data-testid="auth-form">
        <Card.Section bg="blue.6" p="md">
          <Title order={2} c="white">
            {isRegisterTab ? "Create an Account" : "Login to Notably"}
          </Title>
        </Card.Section>

        <Tabs value={activeTab} onChange={setActiveTab} mt="md">
          <Tabs.List grow>
            <Tabs.Tab value="login">Login</Tabs.Tab>
            <Tabs.Tab value="register">Register</Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="login" pt="md">
            <form onSubmit={handleSubmit}>
              <Stack gap="md" pos="relative">
                <LoadingOverlay
                  visible={loading}
                  zIndex={1000}
                  overlayProps={{ radius: "sm", blur: 2 }}
                />

                <TextInput
                  label="Username"
                  placeholder="Username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                  leftSection={<IconUserPlus size={16} />}
                />

                <PasswordInput
                  label="Password"
                  placeholder="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />

                <Button type="submit" fullWidth color="blue" mt="md">
                  Login
                </Button>
              </Stack>
            </form>
          </Tabs.Panel>

          <Tabs.Panel value="register" pt="md">
            <form onSubmit={handleSubmit}>
              <Stack gap="md" pos="relative">
                <LoadingOverlay
                  visible={loading}
                  zIndex={1000}
                  overlayProps={{ radius: "sm", blur: 2 }}
                />

                <TextInput
                  label="Username"
                  placeholder="Username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                  leftSection={<IconUserPlus size={16} />}
                />

                <TextInput
                  label="Email"
                  placeholder="Email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />

                <PasswordInput
                  label="Password"
                  placeholder="Password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />

                <Button type="submit" fullWidth color="blue" mt="md">
                  Register
                </Button>
              </Stack>
            </form>
          </Tabs.Panel>
        </Tabs>
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
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const loadTables = async () => {
    setLoading(true);
    try {
      const res = await client.listTables();
      setTables(res.tables);
    } catch (error: unknown) {
      notifications.show({
        title: "Error",
        message: getErrorMessage(error),
        color: "red",
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTables();
  }, []);

  return (
    <div data-testid="main-app">
      <AppShell header={{ height: 60 }} padding="md">
        <AppShell.Header>
          <Flex justify="space-between" align="center" h="100%" px="md">
            <Group>
              <IconDatabase size={24} />
              <Title order={3}>Notably</Title>
            </Group>
            <Button variant="subtle" color="red" onClick={onLogout} leftSection={<IconLogout size={16} />}>
              Logout
            </Button>
          </Flex>
        </AppShell.Header>

        <AppShell.Main>
          <Container size="lg" py="md">
            <Paper p="md" shadow="sm" radius="md" withBorder>
              <Flex justify="space-between" align="center" mb="md">
                <Title order={4}>Tables</Title>
                <Button leftSection={<IconPlus size={16} />} disabled>
                  Create Table
                </Button>
              </Flex>

              {tables.length === 0 ? (
                <Center py="xl">
                  <Stack align="center" gap="sm">
                    <IconTable size={48} opacity={0.3} />
                    <Text c="dimmed">No tables yet. Create your first table to get started.</Text>
                  </Stack>
                </Center>
              ) : (
                <Table striped highlightOnHover withTableBorder>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Name</Table.Th>
                      <Table.Th>Created</Table.Th>
                      <Table.Th>Columns</Table.Th>
                      <Table.Th>Actions</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {tables.map((table) => (
                      <Table.Tr key={table.name}>
                        <Table.Td>
                          <Text fw={500}>{table.name}</Text>
                        </Table.Td>
                        <Table.Td>{new Date(table.createdAt).toLocaleString()}</Table.Td>
                        <Table.Td>
                          <Group gap="xs">
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
                        </Table.Td>
                        <Table.Td>
                          <Button variant="light" size="xs" disabled>
                            Open
                          </Button>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
              )}
            </Paper>
          </Container>
        </AppShell.Main>
      </AppShell>
    </div>
  );
}

export default App;