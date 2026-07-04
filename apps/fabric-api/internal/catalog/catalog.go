package catalog

type Config struct {
	WorkspaceImage  string
	WorkspaceDomain string
	StorageClass    string
}

type Catalog struct {
	SchemaVersion         int                    `json:"schemaVersion"`
	Owner                 string                 `json:"owner"`
	ProductPresets        []ProductPreset        `json:"productPresets"`
	ComputeProfiles       []ComputeProfile       `json:"computeProfiles"`
	ProviderInstanceTypes []ProviderInstanceType `json:"providerInstanceTypes"`
	CapacityPools         []CapacityPool         `json:"capacityPools"`
	SchedulingPolicies    []SchedulingPolicy     `json:"schedulingPolicies"`
	StorageClasses        []StorageClass         `json:"storageClasses"`
	WorkspaceImages       []WorkspaceImage       `json:"workspaceImages"`
	IngressDomains        []IngressDomain        `json:"ingressDomains"`
}

type ProductPreset struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Accelerator        string `json:"accelerator"`
	DefaultCPU         int    `json:"defaultCpu"`
	DefaultMemoryGB    int    `json:"defaultMemoryGb"`
	DefaultGPU         int    `json:"defaultGpu"`
	DefaultDiskGB      int    `json:"defaultDiskGb"`
	Available          bool   `json:"available"`
	UnavailableReason  string `json:"unavailableReason,omitempty"`
	ComputeProfileID   string `json:"computeProfileId"`
	StorageClassID     string `json:"storageClassId"`
	WorkspaceImageID   string `json:"workspaceImageId"`
	IngressDomainID    string `json:"ingressDomainId"`
	SchedulingPolicyID string `json:"schedulingPolicyId"`
}

type ComputeProfile struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Accelerator         string `json:"accelerator"`
	Provider            string `json:"provider"`
	MinCPU              int    `json:"minCpu"`
	MaxCPU              int    `json:"maxCpu"`
	MinMemoryGB         int    `json:"minMemoryGb"`
	MaxMemoryGB         int    `json:"maxMemoryGb"`
	MaxGPU              int    `json:"maxGpu"`
	CustomShapesAllowed bool   `json:"customShapesAllowed"`
	Available           bool   `json:"available"`
	UnavailableReason   string `json:"unavailableReason,omitempty"`
}

type ProviderInstanceType struct {
	ID              string   `json:"id"`
	Provider        string   `json:"provider"`
	Region          string   `json:"region"`
	Zone            string   `json:"zone,omitempty"`
	Family          string   `json:"family"`
	InstanceType    string   `json:"instanceType"`
	Accelerator     string   `json:"accelerator"`
	MinCPU          int      `json:"minCpu"`
	MaxCPU          int      `json:"maxCpu"`
	MinMemoryGB     int      `json:"minMemoryGb"`
	MaxMemoryGB     int      `json:"maxMemoryGb"`
	MaxGPU          int      `json:"maxGpu"`
	CapacityPoolIDs []string `json:"capacityPoolIds"`
	Available       bool     `json:"available"`
	PriceHint       string   `json:"priceHint,omitempty"`
}

type CapacityPool struct {
	ID                string   `json:"id"`
	Provider          string   `json:"provider"`
	Kind              string   `json:"kind"`
	IsolationMode     string   `json:"isolationMode"`
	Region            string   `json:"region"`
	Zone              string   `json:"zone,omitempty"`
	ProviderRef       string   `json:"providerRef,omitempty"`
	ComputeProfileIDs []string `json:"computeProfileIds"`
	Available         bool     `json:"available"`
	UnavailableReason string   `json:"unavailableReason,omitempty"`
}

type SchedulingPolicy struct {
	ID                  string `json:"id"`
	Mode                string `json:"mode"`
	CapacityPoolKind    string `json:"capacityPoolKind"`
	CreatePoolIfMissing bool   `json:"createPoolIfMissing"`
	RejectIfNoCapacity  bool   `json:"rejectIfNoCapacity"`
	Description         string `json:"description"`
}

type StorageClass struct {
	ID               string `json:"id"`
	Provider         string `json:"provider"`
	StorageClassName string `json:"storageClassName"`
	AccessMode       string `json:"accessMode"`
	Available        bool   `json:"available"`
}

type WorkspaceImage struct {
	ID               string   `json:"id"`
	Image            string   `json:"image"`
	Port             int      `json:"port"`
	PersistentMounts []string `json:"persistentMounts"`
	Available        bool     `json:"available"`
}

type IngressDomain struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	PathPattern string `json:"pathPattern"`
	Available   bool   `json:"available"`
}

func DefaultCatalog(cfg Config) Catalog {
	return Catalog{
		SchemaVersion: 2,
		Owner:         "OPL Fabric",
		ProductPresets: []ProductPreset{
			{ID: "basic", Name: "Basic", Accelerator: "cpu", DefaultCPU: 2, DefaultMemoryGB: 4, DefaultGPU: 0, DefaultDiskGB: 10, Available: true, ComputeProfileID: "cpu-general", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace", SchedulingPolicyID: "shared-pool-first"},
			{ID: "pro", Name: "Pro", Accelerator: "cpu", DefaultCPU: 8, DefaultMemoryGB: 16, DefaultGPU: 0, DefaultDiskGB: 100, Available: true, ComputeProfileID: "cpu-general", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace", SchedulingPolicyID: "shared-pool-first"},
			{ID: "gpu", Name: "GPU", Accelerator: "gpu", DefaultCPU: 16, DefaultMemoryGB: 64, DefaultGPU: 1, DefaultDiskGB: 500, Available: false, UnavailableReason: "gpu_capacity_not_verified", ComputeProfileID: "gpu-general", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace", SchedulingPolicyID: "dedicated-nodepool"},
		},
		ComputeProfiles: []ComputeProfile{
			{ID: "cpu-general", Name: "CPU General", Accelerator: "cpu", Provider: "tencent-tke", MinCPU: 1, MaxCPU: 64, MinMemoryGB: 1, MaxMemoryGB: 256, MaxGPU: 0, CustomShapesAllowed: true, Available: true},
			{ID: "gpu-general", Name: "GPU General", Accelerator: "gpu", Provider: "tencent-tke", MinCPU: 4, MaxCPU: 128, MinMemoryGB: 16, MaxMemoryGB: 1024, MaxGPU: 8, CustomShapesAllowed: true, Available: false, UnavailableReason: "gpu_capacity_not_verified"},
		},
		ProviderInstanceTypes: []ProviderInstanceType{
			{ID: "tencent-cpu-dynamic", Provider: "tencent-cloud", Region: "config:TENCENT_TKE_REGION", Family: "S/C/M series", InstanceType: "resolved-by-tencent-sdk", Accelerator: "cpu", MinCPU: 1, MaxCPU: 64, MinMemoryGB: 1, MaxMemoryGB: 256, MaxGPU: 0, CapacityPoolIDs: []string{"shared-cpu", "dedicated-nodepool-template"}, Available: true},
			{ID: "tencent-gpu-dynamic", Provider: "tencent-cloud", Region: "config:TENCENT_TKE_REGION", Family: "GPU series", InstanceType: "resolved-by-tencent-sdk", Accelerator: "gpu", MinCPU: 4, MaxCPU: 128, MinMemoryGB: 16, MaxMemoryGB: 1024, MaxGPU: 8, CapacityPoolIDs: []string{"dedicated-nodepool-template"}, Available: false},
		},
		CapacityPools: []CapacityPool{
			{ID: "shared-cpu", Provider: "tencent-tke", Kind: "shared", IsolationMode: "shared_pool", Region: "config:TENCENT_TKE_REGION", ComputeProfileIDs: []string{"cpu-general"}, Available: true},
			{ID: "dedicated-nodepool-template", Provider: "tencent-tke", Kind: "dedicated_template", IsolationMode: "dedicated_nodepool", Region: "config:TENCENT_TKE_REGION", ComputeProfileIDs: []string{"cpu-general", "gpu-general"}, Available: true},
		},
		SchedulingPolicies: []SchedulingPolicy{
			{ID: "shared-pool-first", Mode: "shared_pool", CapacityPoolKind: "shared", CreatePoolIfMissing: false, RejectIfNoCapacity: true, Description: "Use existing shared capacity that matches the requested custom shape."},
			{ID: "dedicated-nodepool", Mode: "dedicated_nodepool", CapacityPoolKind: "dedicated_template", CreatePoolIfMissing: true, RejectIfNoCapacity: true, Description: "Create or reuse an isolated Tencent TKE node pool for the workspace shape."},
		},
		StorageClasses:  []StorageClass{{ID: "workspace-cbs", Provider: "tencent-tke", StorageClassName: cfg.StorageClass, AccessMode: "ReadWriteOnce", Available: true}},
		WorkspaceImages: []WorkspaceImage{{ID: "one-person-lab-app", Image: cfg.WorkspaceImage, Port: 3000, PersistentMounts: []string{"/data", "/projects"}, Available: true}},
		IngressDomains:  []IngressDomain{{ID: "workspace", Host: cfg.WorkspaceDomain, PathPattern: "/w/<workspaceId>", Available: true}},
	}
}
