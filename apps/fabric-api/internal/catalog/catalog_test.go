package catalog

import "testing"

func TestDefaultCatalogPackages(t *testing.T) {
	catalog := DefaultCatalog(Config{
		WorkspaceImage:  "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
		WorkspaceDomain: "workspace.medopl.cn",
		StorageClass:    "cbs",
	})

	if len(catalog.WorkspacePackages) != 3 {
		t.Fatalf("workspace package count = %d", len(catalog.WorkspacePackages))
	}

	basic := catalog.WorkspacePackages[0]
	if basic.ID != "basic" || !basic.Available || basic.CPU != 2 || basic.MemoryGB != 4 || basic.DiskGB != 10 {
		t.Fatalf("basic package mismatch: %+v", basic)
	}

	gpu := catalog.WorkspacePackages[2]
	if gpu.ID != "gpu" || gpu.Available || gpu.UnavailableReason != "gpu_node_pool_not_verified" {
		t.Fatalf("gpu package mismatch: %+v", gpu)
	}

	for _, profile := range catalog.ComputeProfiles {
		if profile.Accelerator == "" {
			t.Fatalf("compute profile %s missing accelerator", profile.ID)
		}
	}
}
