import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Cpu,
  Database,
  HardDrive,
  Package,
  RefreshCw,
  Server,
  ShieldAlert,
  Wrench
} from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { fetchReadiness, type Catalog, type Readiness, type WorkspacePackage } from "./api";

type LoadState =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "loaded"; readiness: Readiness };

function App() {
  const [state, setState] = useState<LoadState>({ status: "loading" });

  const loadReadiness = () => {
    setState({ status: "loading" });
    fetchReadiness()
      .then((readiness) => setState({ status: "loaded", readiness }))
      .catch((error: unknown) => {
        setState({
          status: "error",
          message: error instanceof Error ? error.message : "Fabric readiness request failed."
        });
      });
  };

  useEffect(() => {
    loadReadiness();
  }, []);

  if (state.status === "loading") {
    return (
      <main className="shell">
        <Header onRefresh={loadReadiness} refreshDisabled />
        <section className="status-panel loading-panel">
          <RefreshCw className="spin" size={22} aria-hidden="true" />
          <div>
            <p className="eyebrow">Fabric readiness</p>
            <h1>Loading operator state</h1>
          </div>
        </section>
        <SkeletonGrid />
      </main>
    );
  }

  if (state.status === "error") {
    return (
      <main className="shell">
        <Header onRefresh={loadReadiness} />
        <section className="status-panel error-panel">
          <ShieldAlert size={24} aria-hidden="true" />
          <div>
            <p className="eyebrow">Fabric readiness</p>
            <h1>Console unavailable</h1>
            <p className="error-copy">{state.message}</p>
          </div>
        </section>
      </main>
    );
  }

  return (
    <main className="shell">
      <Header onRefresh={loadReadiness} />
      <ReadinessSummary readiness={state.readiness} />
      <OperationsGrid readiness={state.readiness} />
      <WorkspacePackages packages={state.readiness.resourceCatalog.workspacePackages} />
      <CatalogSummary catalog={state.readiness.resourceCatalog} />
    </main>
  );
}

function Header({ onRefresh, refreshDisabled = false }: { onRefresh: () => void; refreshDisabled?: boolean }) {
  return (
    <header className="topbar">
      <div>
        <p className="eyebrow">OPL Fabric</p>
        <h1>Operator Console</h1>
      </div>
      <button className="icon-button" type="button" onClick={onRefresh} disabled={refreshDisabled} aria-label="Refresh readiness">
        <RefreshCw size={17} aria-hidden="true" />
      </button>
    </header>
  );
}

function ReadinessSummary({ readiness }: { readiness: Readiness }) {
  const statusLabel = readiness.ready ? "Ready" : "Blocked";
  const statusIcon = readiness.ready ? <CheckCircle2 size={22} aria-hidden="true" /> : <AlertTriangle size={22} aria-hidden="true" />;

  return (
    <section className={`status-panel ${readiness.ready ? "ready-panel" : "blocked-panel"}`}>
      <div className="status-icon">{statusIcon}</div>
      <div className="status-copy">
        <p className="eyebrow">Readiness</p>
        <h2>{statusLabel}</h2>
        <dl className="metric-strip">
          <Metric label="Provider" value={readiness.provider || "unknown"} />
          <Metric label="Missing env" value={readiness.missingEnv.length.toString()} />
          <Metric label="Blockers" value={readiness.blockers.length.toString()} />
          <Metric label="Packages" value={readiness.resourceCatalog.workspacePackages.length.toString()} />
        </dl>
      </div>
    </section>
  );
}

function OperationsGrid({ readiness }: { readiness: Readiness }) {
  return (
    <section className="ops-grid" aria-label="Readiness details">
      <DetailPanel title="Missing Env" icon={<Database size={17} aria-hidden="true" />} empty="None reported" items={readiness.missingEnv} />
      <DetailPanel title="Blockers" icon={<AlertTriangle size={17} aria-hidden="true" />} empty="No active blockers" items={readiness.blockers} tone="blocked" />
      <DetailPanel title="Repair Hints" icon={<Wrench size={17} aria-hidden="true" />} empty="No repair hints" items={readiness.repairHints} />
    </section>
  );
}

function DetailPanel({
  title,
  icon,
  empty,
  items,
  tone
}: {
  title: string;
  icon: ReactNode;
  empty: string;
  items: string[];
  tone?: "blocked";
}) {
  return (
    <section className={`panel ${tone === "blocked" ? "panel-alert" : ""}`}>
      <div className="panel-heading">
        {icon}
        <h3>{title}</h3>
        <span>{items.length}</span>
      </div>
      {items.length === 0 ? (
        <p className="empty">{empty}</p>
      ) : (
        <ul className="compact-list">
          {items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      )}
    </section>
  );
}

function WorkspacePackages({ packages }: { packages: WorkspacePackage[] }) {
  return (
    <section className="panel workspace-panel">
      <div className="panel-heading">
        <Package size={17} aria-hidden="true" />
        <h3>Workspace Packages</h3>
        <span>{packages.length}</span>
      </div>
      <div className="package-table" role="table" aria-label="Workspace packages">
        <div className="package-row table-head" role="row">
          <span role="columnheader">Package</span>
          <span role="columnheader">Shape</span>
          <span role="columnheader">Storage</span>
          <span role="columnheader">Refs</span>
          <span role="columnheader">State</span>
        </div>
        {packages.map((item) => (
          <PackageRow key={item.id} item={item} />
        ))}
      </div>
    </section>
  );
}

function PackageRow({ item }: { item: WorkspacePackage }) {
  return (
    <div className="package-row" role="row">
      <div className="package-name" role="cell">
        <strong>{item.name}</strong>
        <span>{item.id}</span>
      </div>
      <div role="cell">
        <span className="mono">{item.server}</span>
        <small>
          {item.cpu} CPU / {item.memoryGb} GB / {item.gpu} GPU
        </small>
      </div>
      <div role="cell">
        <span>{item.diskGb} GB</span>
        <small>{item.storageClassId}</small>
      </div>
      <div role="cell">
        <span>{item.computeProfileId}</span>
        <small>{item.workspaceImageId}</small>
      </div>
      <div role="cell">
        <span className={`pill ${item.available ? "pill-ok" : "pill-warn"}`}>{item.available ? "Available" : "Unavailable"}</span>
        {!item.available && <small>{item.unavailableReason ?? "unavailable"}</small>}
      </div>
    </div>
  );
}

function CatalogSummary({ catalog }: { catalog: Catalog }) {
  const unavailable = useMemo(
    () => [
      ...catalog.computeProfiles.filter((item) => !item.available).map((item) => `compute:${item.id}`),
      ...catalog.storageClasses.filter((item) => !item.available).map((item) => `storage:${item.id}`),
      ...catalog.workspaceImages.filter((item) => !item.available).map((item) => `image:${item.id}`),
      ...catalog.ingressDomains.filter((item) => !item.available).map((item) => `ingress:${item.id}`)
    ],
    [catalog]
  );

  return (
    <section className="catalog-grid" aria-label="Resource catalog">
      <CatalogTile icon={<Cpu size={18} aria-hidden="true" />} label="Compute" count={catalog.computeProfiles.length} />
      <CatalogTile icon={<HardDrive size={18} aria-hidden="true" />} label="Storage" count={catalog.storageClasses.length} />
      <CatalogTile icon={<Server size={18} aria-hidden="true" />} label="Images" count={catalog.workspaceImages.length} />
      <CatalogTile icon={<Activity size={18} aria-hidden="true" />} label="Ingress" count={catalog.ingressDomains.length} />
      <section className="panel unavailable-panel">
        <div className="panel-heading">
          <AlertTriangle size={17} aria-hidden="true" />
          <h3>Unavailable Catalog Refs</h3>
          <span>{unavailable.length}</span>
        </div>
        {unavailable.length === 0 ? (
          <p className="empty">None reported</p>
        ) : (
          <ul className="compact-list">
            {unavailable.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        )}
      </section>
    </section>
  );
}

function CatalogTile({ icon, label, count }: { icon: ReactNode; label: string; count: number }) {
  return (
    <section className="catalog-tile">
      {icon}
      <span>{label}</span>
      <strong>{count}</strong>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function SkeletonGrid() {
  return (
    <section className="ops-grid" aria-label="Loading placeholders">
      <div className="skeleton" />
      <div className="skeleton" />
      <div className="skeleton" />
    </section>
  );
}

export default App;
