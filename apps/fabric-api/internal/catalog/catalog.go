package catalog

type Config struct {
	WorkspaceImage  string
	WorkspaceDomain string
	StorageClass    string
}

type Catalog struct {
	SchemaVersion     int                `json:"schemaVersion"`
	Owner             string             `json:"owner"`
	WorkspacePackages []WorkspacePackage `json:"workspacePackages"`
	ComputeProfiles   []ComputeProfile   `json:"computeProfiles"`
	StorageClasses    []StorageClass     `json:"storageClasses"`
	WorkspaceImages   []WorkspaceImage   `json:"workspaceImages"`
	IngressDomains    []IngressDomain    `json:"ingressDomains"`
}

type WorkspacePackage struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Accelerator       string `json:"accelerator"`
	CPU               int    `json:"cpu"`
	MemoryGB          int    `json:"memoryGb"`
	GPU               int    `json:"gpu"`
	Server            string `json:"server"`
	DiskGB            int    `json:"diskGb"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
	ComputeProfileID  string `json:"computeProfileId"`
	StorageClassID    string `json:"storageClassId"`
	WorkspaceImageID  string `json:"workspaceImageId"`
	IngressDomainID   string `json:"ingressDomainId"`
}

type ComputeProfile struct {
	ID          string `json:"id"`
	Accelerator string `json:"accelerator"`
	Provider    string `json:"provider"`
	CPU         int    `json:"cpu"`
	MemoryGB    int    `json:"memoryGb"`
	GPU         int    `json:"gpu"`
	Available   bool   `json:"available"`
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
		SchemaVersion: 1,
		Owner:         "OPL Fabric",
		WorkspacePackages: []WorkspacePackage{
			{ID: "basic", Name: "Basic Workspace", Accelerator: "cpu", CPU: 2, MemoryGB: 4, GPU: 0, Server: "2c4g", DiskGB: 10, Available: true, ComputeProfileID: "cpu-basic", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
			{ID: "pro", Name: "Pro Workspace", Accelerator: "cpu", CPU: 8, MemoryGB: 16, GPU: 0, Server: "8c16g", DiskGB: 100, Available: true, ComputeProfileID: "cpu-pro", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
			{ID: "gpu", Name: "GPU Workspace", Accelerator: "gpu", CPU: 16, MemoryGB: 64, GPU: 1, Server: "16c64g-1gpu", DiskGB: 500, Available: false, UnavailableReason: "gpu_node_pool_not_verified", ComputeProfileID: "gpu-standard", StorageClassID: "workspace-cbs", WorkspaceImageID: "one-person-lab-app", IngressDomainID: "workspace"},
		},
		ComputeProfiles: []ComputeProfile{
			{ID: "cpu-basic", Accelerator: "cpu", Provider: "tencent-tke", CPU: 2, MemoryGB: 4, GPU: 0, Available: true},
			{ID: "cpu-pro", Accelerator: "cpu", Provider: "tencent-tke", CPU: 8, MemoryGB: 16, GPU: 0, Available: true},
			{ID: "gpu-standard", Accelerator: "gpu", Provider: "tencent-tke", CPU: 16, MemoryGB: 64, GPU: 1, Available: false},
		},
		StorageClasses:  []StorageClass{{ID: "workspace-cbs", Provider: "tencent-tke", StorageClassName: cfg.StorageClass, AccessMode: "ReadWriteOnce", Available: true}},
		WorkspaceImages: []WorkspaceImage{{ID: "one-person-lab-app", Image: cfg.WorkspaceImage, Port: 3000, PersistentMounts: []string{"/data", "/projects"}, Available: true}},
		IngressDomains:  []IngressDomain{{ID: "workspace", Host: cfg.WorkspaceDomain, PathPattern: "/w/<workspaceId>", Available: true}},
	}
}
