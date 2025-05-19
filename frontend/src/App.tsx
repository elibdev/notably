import React, { useState, useEffect } from "react";
import { ApiClient, TableInfo, RowData, RowEvent } from "./api";

export function App() {
  const [apiKey, setApiKey] = useState<string>(localStorage.getItem("apiKey") || "");
  const [client, setClient] = useState<ApiClient | null>(null);
  const [view, setView] = useState<"login" | "register" | "tables">(apiKey ? "tables" : "login");

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
    setView("login");
  };

  if (!client) {
    if (view === "login") {
      return (
        <AuthForm
          title="Login"
          onSubmit={async ({ username, password }) => {
            const res = await ApiClient.login(username, password);
            setApiKey(res.apiKey);
            localStorage.setItem("apiKey", res.apiKey);
            setView("tables");
          }}
          onSwitch={() => setView("register")}
        />
      );
    }
    return (
      <AuthForm
        title="Register"
        includeEmail
        onSubmit={async ({ username, email, password }) => {
          const res = await ApiClient.register(username, email!, password);
          setApiKey(res.apiKey);
          localStorage.setItem("apiKey", res.apiKey);
          setView("tables");
        }}
        onSwitch={() => setView("login")}
      />
    );
  }
  return <MainApp client={client} onLogout={handleLogout} />;
}

interface AuthFormProps {
  title: string;
  includeEmail?: boolean;
  onSubmit: (fields: { username: string; email?: string; password: string }) => Promise<void>;
  onSwitch: () => void;
}

function AuthForm({ title, includeEmail, onSubmit, onSwitch }: AuthFormProps) {
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await onSubmit({ username, email, password });
    } catch (err: any) {
      setError(err.message);
    }
  };

  return (
    <div style={{ maxWidth: 400, margin: "auto", padding: 20 }}>
      <h2>{title}</h2>
      <form onSubmit={handleSubmit}>
        <div>
          <input
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </div>
        {includeEmail && (
          <div>
            <input
              placeholder="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
        )}
        <div>
          <input
            placeholder="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </div>
        <div>
          <button type="submit">{title}</button>
        </div>
        {error && <div style={{ color: "red" }}>{error}</div>}
      </form>
      <div>
        <button onClick={onSwitch}>{includeEmail ? "Have an account? Login" : "Register"}</button>
      </div>
    </div>
  );
}

interface MainAppProps {
  client: ApiClient;
  onLogout: () => void;
}

function MainApp({ client, onLogout }: MainAppProps) {
  const [tables, setTables] = useState<TableInfo[]>([]);
  const [newTableName, setNewTableName] = useState("");
  const [selectedTable, setSelectedTable] = useState<string | null>(null);

  const loadTables = async () => {
    try {
      const res = await client.listTables();
      setTables(res.tables);
    } catch (err: any) {
      alert(err.message);
    }
  };

  useEffect(() => {
    loadTables();
  }, []);

  if (selectedTable) {
    return (
      <TableView table={selectedTable} client={client} onBack={() => setSelectedTable(null)} />
    );
  }
  return (
    <div style={{ padding: 20 }}>
      <h2>Your Tables</h2>
      <button onClick={onLogout}>Logout</button>
      <ul>
        {tables.map((t) => (
          <li key={t.name}>
            <button onClick={() => setSelectedTable(t.name)}>{t.name}</button> (created at{" "}
            {t.createdAt})
          </li>
        ))}
      </ul>
      <div>
        <h3>Create Table</h3>
        <input
          value={newTableName}
          onChange={(e) => setNewTableName(e.target.value)}
          placeholder="Table name"
        />
        <button
          onClick={async () => {
            try {
              await client.createTable(newTableName);
              setNewTableName("");
              loadTables();
            } catch (err: any) {
              alert(err.message);
            }
          }}
        >
          Create
        </button>
      </div>
    </div>
  );
}

interface TableViewProps {
  table: string;
  client: ApiClient;
  onBack: () => void;
}

function TableView({ table, client, onBack }: TableViewProps) {
  const [rows, setRows] = useState<RowData[]>([]);
  const [newId, setNewId] = useState("");
  const [newValues, setNewValues] = useState("{}");
  const [snapshotAt, setSnapshotAt] = useState("");
  const [historyRange, setHistoryRange] = useState({ start: "", end: "" });
  const [historyEvents, setHistoryEvents] = useState<RowEvent[]>([]);

  const loadRows = async () => {
    try {
      const res = await client.listRows(table);
      setRows(res.rows);
    } catch (err: any) {
      alert(err.message);
    }
  };

  useEffect(() => {
    loadRows();
  }, [table]);

  const applySnapshot = async () => {
    try {
      const atIso = snapshotAt ? new Date(snapshotAt).toISOString() : undefined;
      const res = await client.snapshot(table, atIso);
      setRows(res.rows);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const loadHistory = async () => {
    try {
      const startIso = new Date(historyRange.start).toISOString();
      const endIso = new Date(historyRange.end).toISOString();
      const e = await client.history(table, startIso, endIso);
      setHistoryEvents(e.events);
    } catch (err: any) {
      alert(err.message);
    }
  };

  const addRow = async () => {
    try {
      const vals = JSON.parse(newValues);
      await client.createRow(table, newId, vals);
      setNewId("");
      setNewValues("{}");
      loadRows();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const updateRow = async (id: string) => {
    const current = rows.find((r) => r.id === id)?.values;
    const input = prompt("Enter new JSON values", JSON.stringify(current, null, 2));
    if (!input) return;
    try {
      const vals = JSON.parse(input);
      await client.updateRow(table, id, vals);
      loadRows();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const deleteRow = async (id: string) => {
    if (!window.confirm(`Delete row ${id}?`)) return;
    try {
      await client.deleteRow(table, id);
      loadRows();
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div style={{ padding: 20 }}>
      <button onClick={onBack}>Back to Tables</button>
      <h2>Table: {table}</h2>
      <h3>Rows</h3>
      <ul>
        {rows.map((r) => (
          <li key={r.id}>
            <strong>{r.id}</strong> [at {r.timestamp}] {JSON.stringify(r.values)}{" "}
            <button onClick={() => updateRow(r.id)}>Edit</button>{" "}
            <button onClick={() => deleteRow(r.id)}>Delete</button>
          </li>
        ))}
      </ul>
      <div>
        <h4>Add Row</h4>
        <input value={newId} onChange={(e) => setNewId(e.target.value)} placeholder="Row ID" />
        <input
          value={newValues}
          onChange={(e) => setNewValues(e.target.value)}
          placeholder="Values JSON"
          style={{ width: "60%" }}
        />
        <button onClick={addRow}>Add</button>
      </div>
      <div>
        <h4>Snapshot</h4>
        <input
          type="datetime-local"
          value={snapshotAt}
          onChange={(e) => setSnapshotAt(e.target.value)}
        />
        <button onClick={applySnapshot}>Load Snapshot</button>
      </div>
      <div>
        <h4>History</h4>
        <label>
          Start:
          <input
            type="datetime-local"
            value={historyRange.start}
            onChange={(e) => setHistoryRange((prev) => ({ ...prev, start: e.target.value }))}
          />
        </label>
        <label>
          End:
          <input
            type="datetime-local"
            value={historyRange.end}
            onChange={(e) => setHistoryRange((prev) => ({ ...prev, end: e.target.value }))}
          />
        </label>
        <button onClick={loadHistory}>Load History</button>
      </div>
      <ul>
        {historyEvents.map((ev) => (
          <li key={`${ev.id}-${ev.timestamp}`}>
            <strong>{ev.id}</strong> [at {ev.timestamp}] {JSON.stringify(ev.values)}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;
