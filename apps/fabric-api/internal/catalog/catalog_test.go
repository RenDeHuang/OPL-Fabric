package catalog

import "testing"

func TestDefaultCatalogUsesPresetsAndCustomShapeProfiles(t *testing.T) {
	catalog := DefaultCatalog(Config{
		WorkspaceImage:  "ghcr.io/gaofeng21cn/one-person-lab-app:latest",
		WorkspaceDomain: "workspace.medopl.cn",
		StorageClass:    "cbs",
	})

	if catalog.SchemaVersion != 2 {
		t.Fatalf("schema version = %d, want 2", catalog.SchemaVersion)
	}
	if len(catalog.ProductPresets) != 3 {
		t.Fatalf("product preset count = %d", len(catalog.ProductPresets))
	}

	basic := catalog.ProductPresets[0]
	if basic.ID != "basic" || !basic.Available || basic.DefaultCPU != 2 || basic.DefaultMemoryGB != 4 || basic.DefaultDiskGB != 10 {
		t.Fatalf("basic preset mismatch: %+v", basic)
	}
	if basic.ComputeProfileID != "cpu-general" || basic.SchedulingPolicyID != "shared-pool-first" {
		t.Fatalf("basic preset should reference general CPU policy: %+v", basic)
	}

	gpu := catalog.ProductPresets[2]
	if gpu.ID != "gpu" || gpu.Available || gpu.UnavailableReason != "gpu_capacity_not_verified" {
		t.Fatalf("gpu preset mismatch: %+v", gpu)
	}

	for _, profile := range catalog.ComputeProfiles {
		if profile.Accelerator == "" {
			t.Fatalf("compute profile %s missing accelerator", profile.ID)
		}
		if !profile.CustomShapesAllowed {
			t.Fatalf("compute profile %s must allow custom shapes", profile.ID)
		}
	}

	if len(catalog.ProviderInstanceTypes) == 0 {
		t.Fatal("provider instance type resolver entries are required")
	}
	if len(catalog.CapacityPools) == 0 {
		t.Fatal("capacity pools are required")
	}
	if len(catalog.SchedulingPolicies) != 2 {
		t.Fatalf("scheduling policy count = %d, want 2", len(catalog.SchedulingPolicies))
	}
}
