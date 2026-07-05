package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

func TestRunOnceLeasesAndAppliesAcceptedWorkspaceOperation(t *testing.T) {
	store := &fakeStore{
		operations: []postgres.OperationRow{{ID: "op-workspace", ResourceKind: "workspace", ResourceID: "ws-1", State: "accepted"}},
		leases:     map[string]bool{"op-workspace": true},
	}
	orch := &fakeOrchestrator{}
	w := Worker{Store: store, Orchestrator: orch, Owner: "worker-1", BatchSize: 10, LeaseTTL: time.Minute}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(orch.applied) != 1 || orch.applied[0] != "op-workspace" {
		t.Fatalf("applied = %#v", orch.applied)
	}
	if len(store.failures) != 0 {
		t.Fatalf("failures = %#v", store.failures)
	}
}

func TestRunOnceSkipsOperationWhenLeaseIsNotAcquired(t *testing.T) {
	store := &fakeStore{
		operations: []postgres.OperationRow{{ID: "op-1", ResourceKind: "storage_volume", ResourceID: "storage-1", State: "accepted"}},
		leases:     map[string]bool{"op-1": false},
	}
	orch := &fakeOrchestrator{}
	w := Worker{Store: store, Orchestrator: orch, Owner: "worker-1", BatchSize: 10, LeaseTTL: time.Minute}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(orch.applied) != 0 {
		t.Fatalf("applied = %#v, want none", orch.applied)
	}
}

func TestRunOnceRecordsFailureWhenApplyFails(t *testing.T) {
	applyErr := errors.New("apply_failed")
	store := &fakeStore{
		operations: []postgres.OperationRow{{ID: "op-1", ResourceKind: "compute_allocation", ResourceID: "compute-1", State: "accepted"}},
		leases:     map[string]bool{"op-1": true},
	}
	orch := &fakeOrchestrator{err: applyErr}
	w := Worker{Store: store, Orchestrator: orch, Owner: "worker-1", BatchSize: 10, LeaseTTL: time.Minute}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(store.failures) != 1 || !errors.Is(store.failures["op-1"], applyErr) {
		t.Fatalf("failures = %#v", store.failures)
	}
}

func TestRunOnceProcessesMultipleAcceptedResourceOperations(t *testing.T) {
	store := &fakeStore{
		operations: []postgres.OperationRow{
			{ID: "op-storage", ResourceKind: "storage_volume", ResourceID: "storage-1", State: "accepted"},
			{ID: "op-compute", ResourceKind: "compute_allocation", ResourceID: "compute-1", State: "accepted"},
		},
		leases: map[string]bool{"op-storage": true, "op-compute": true},
	}
	orch := &fakeOrchestrator{}
	w := Worker{Store: store, Orchestrator: orch, Owner: "worker-1", BatchSize: 10, LeaseTTL: time.Minute}

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if len(orch.applied) != 2 || orch.applied[0] != "op-storage" || orch.applied[1] != "op-compute" {
		t.Fatalf("applied = %#v", orch.applied)
	}
}

type fakeStore struct {
	operations []postgres.OperationRow
	leases     map[string]bool
	failures   map[string]error
}

func (s *fakeStore) ListAcceptedOperations(_ context.Context, _ int) ([]postgres.OperationRow, error) {
	return s.operations, nil
}

func (s *fakeStore) LeaseOperation(_ context.Context, id, _ string, _ time.Duration) (bool, error) {
	return s.leases[id], nil
}

func (s *fakeStore) RecordOperationFailure(_ context.Context, id string, err error) error {
	if s.failures == nil {
		s.failures = map[string]error{}
	}
	s.failures[id] = err
	return nil
}

type fakeOrchestrator struct {
	applied []string
	err     error
}

func (o *fakeOrchestrator) Apply(_ context.Context, id string) (orchestrator.Receipt, error) {
	if o.err != nil {
		return orchestrator.Receipt{}, o.err
	}
	o.applied = append(o.applied, id)
	return orchestrator.Receipt{OperationID: id, State: "succeeded"}, nil
}
