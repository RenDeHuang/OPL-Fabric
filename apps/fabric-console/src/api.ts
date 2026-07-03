export interface Readiness {
  ready: boolean;
  provider: string;
  missingEnv: string[];
  resourceCatalog: Catalog;
  blockers: string[];
  repairHints: string[];
}

export interface Catalog {
  schemaVersion: number;
  owner: string;
  workspacePackages: WorkspacePackage[];
  computeProfiles: ComputeProfile[];
  storageClasses: StorageClass[];
  workspaceImages: WorkspaceImage[];
  ingressDomains: IngressDomain[];
}

export interface WorkspacePackage {
  id: string;
  name: string;
  accelerator: string;
  cpu: number;
  memoryGb: number;
  gpu: number;
  server: string;
  diskGb: number;
  available: boolean;
  unavailableReason?: string;
  computeProfileId: string;
  storageClassId: string;
  workspaceImageId: string;
  ingressDomainId: string;
}

export interface ComputeProfile {
  id: string;
  accelerator: string;
  provider: string;
  cpu: number;
  memoryGb: number;
  gpu: number;
  available: boolean;
}

export interface StorageClass {
  id: string;
  provider: string;
  storageClassName: string;
  accessMode: string;
  available: boolean;
}

export interface WorkspaceImage {
  id: string;
  image: string;
  port: number;
  persistentMounts: string[];
  available: boolean;
}

export interface IngressDomain {
  id: string;
  host: string;
  pathPattern: string;
  available: boolean;
}

export class FabricApiError extends Error {
  readonly status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = "FabricApiError";
    this.status = status;
  }
}

function operatorToken(): string {
  const token = import.meta.env.VITE_OPL_OPERATOR_TOKEN;
  if (!token) {
    throw new FabricApiError("VITE_OPL_OPERATOR_TOKEN is not configured for this operator console.");
  }
  return token;
}

export async function fetchReadiness(): Promise<Readiness> {
  const response = await fetch("/api/fabric/readiness", {
    headers: {
      Authorization: `Bearer ${operatorToken()}`
    }
  });

  if (!response.ok) {
    throw new FabricApiError(`Fabric readiness request failed with status ${response.status}.`, response.status);
  }

  return response.json() as Promise<Readiness>;
}
