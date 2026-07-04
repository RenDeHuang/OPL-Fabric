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
  productPresets: ProductPreset[];
  computeProfiles: ComputeProfile[];
  providerInstanceTypes: ProviderInstanceType[];
  capacityPools: CapacityPool[];
  schedulingPolicies: SchedulingPolicy[];
  storageClasses: StorageClass[];
  workspaceImages: WorkspaceImage[];
  ingressDomains: IngressDomain[];
}

export interface ProductPreset {
  id: string;
  name: string;
  accelerator: string;
  defaultCpu: number;
  defaultMemoryGb: number;
  defaultGpu: number;
  defaultDiskGb: number;
  available: boolean;
  unavailableReason?: string;
  computeProfileId: string;
  storageClassId: string;
  workspaceImageId: string;
  ingressDomainId: string;
  schedulingPolicyId: string;
}

export interface ComputeProfile {
  id: string;
  name: string;
  accelerator: string;
  provider: string;
  minCpu: number;
  maxCpu: number;
  minMemoryGb: number;
  maxMemoryGb: number;
  maxGpu: number;
  customShapesAllowed: boolean;
  available: boolean;
  unavailableReason?: string;
}

export interface ProviderInstanceType {
  id: string;
  provider: string;
  region: string;
  zone?: string;
  family: string;
  instanceType: string;
  accelerator: string;
  minCpu: number;
  maxCpu: number;
  minMemoryGb: number;
  maxMemoryGb: number;
  maxGpu: number;
  capacityPoolIds: string[];
  available: boolean;
  priceHint?: string;
}

export interface CapacityPool {
  id: string;
  provider: string;
  kind: string;
  isolationMode: string;
  region: string;
  zone?: string;
  providerRef?: string;
  computeProfileIds: string[];
  available: boolean;
  unavailableReason?: string;
}

export interface SchedulingPolicy {
  id: string;
  mode: string;
  capacityPoolKind: string;
  createPoolIfMissing: boolean;
  rejectIfNoCapacity: boolean;
  description: string;
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

  constructor(message: string, status?: number, cause?: unknown) {
    super(message, { cause });
    this.name = "FabricApiError";
    this.status = status;
  }
}

export async function fetchReadiness(): Promise<Readiness> {
  let response: Response;
  try {
    response = await fetch("/api/fabric/readiness");
  } catch (error) {
    throw new FabricApiError("Fabric API is unreachable. Check the console proxy or API service.", undefined, error);
  }

  if (!response.ok) {
    throw new FabricApiError(readinessErrorMessage(response.status), response.status);
  }

  return response.json() as Promise<Readiness>;
}

function readinessErrorMessage(status: number): string {
  if (status === 401) {
    return "Fabric API rejected the operator session. Check server-side operator token configuration.";
  }
  if (status === 403) {
    return "Fabric API denied the operator session.";
  }
  return `Fabric readiness request failed with status ${status}.`;
}
