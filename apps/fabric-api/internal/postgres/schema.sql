CREATE TABLE IF NOT EXISTS compute_resources (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  product_preset_id TEXT NOT NULL DEFAULT '',
  compute_shape_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  provider_instance_type TEXT NOT NULL DEFAULT '',
  capacity_pool_id TEXT NOT NULL DEFAULT '',
  isolation_mode TEXT NOT NULL DEFAULT '',
  node_pool_id TEXT NOT NULL DEFAULT '',
  runtime_ref TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS storage_volumes (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  product_preset_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  size_gb INTEGER NOT NULL CHECK (size_gb > 0),
  retained BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS storage_attachments (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  compute_id TEXT NOT NULL REFERENCES compute_resources(id),
  storage_id TEXT NOT NULL REFERENCES storage_volumes(id),
  state TEXT NOT NULL,
  mount_path TEXT NOT NULL,
  provider_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_entries (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  attachment_id TEXT NOT NULL REFERENCES storage_attachments(id),
  state TEXT NOT NULL,
  host TEXT NOT NULL,
  path TEXT NOT NULL,
  service_ref TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_operations (
  id TEXT PRIMARY KEY,
  correlation_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  requested_by TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  resource_kind TEXT NOT NULL,
  state TEXT NOT NULL,
  lease_owner TEXT NOT NULL DEFAULT '',
  lease_expires_at TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  provider_refs JSONB NOT NULL DEFAULT '{}'::jsonb,
  evidence_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspaces (
  id TEXT PRIMARY KEY,
  owner_account_id TEXT NOT NULL,
  workspace_name TEXT NOT NULL,
  product_preset_id TEXT NOT NULL DEFAULT '',
  storage_id TEXT NOT NULL REFERENCES storage_volumes(id),
  compute_id TEXT NOT NULL REFERENCES compute_resources(id),
  attachment_id TEXT NOT NULL REFERENCES storage_attachments(id),
  entry_id TEXT NOT NULL REFERENCES workspace_entries(id),
  operation_id TEXT NOT NULL REFERENCES fabric_operations(id),
  state TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_events (
  id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL REFERENCES fabric_operations(id),
  event_name TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fabric_evidence_refs (
  id TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL REFERENCES fabric_operations(id),
  kind TEXT NOT NULL,
  ref TEXT NOT NULL,
  sha256 TEXT NOT NULL CHECK (sha256 ~ '^[A-Fa-f0-9]{64}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS human_gates (
  id TEXT PRIMARY KEY,
  gate_kind TEXT NOT NULL,
  resource_ref TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT PRIMARY KEY,
  operation_id TEXT NOT NULL UNIQUE REFERENCES fabric_operations(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
