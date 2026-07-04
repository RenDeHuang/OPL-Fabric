package worker

import (
	"context"
	"time"

	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/orchestrator"
	"github.com/RenDeHuang/OPL-Fabric/apps/fabric-api/internal/postgres"
)

type Store interface {
	ListAcceptedOperations(context.Context, int) ([]postgres.OperationRow, error)
	LeaseOperation(context.Context, string, string, time.Duration) (bool, error)
	RecordOperationFailure(context.Context, string, error) error
}

type Orchestrator interface {
	Apply(context.Context, string) (orchestrator.Receipt, error)
}

type Worker struct {
	Store        Store
	Orchestrator Orchestrator
	Owner        string
	Interval     time.Duration
	LeaseTTL     time.Duration
	BatchSize    int
}

func (w Worker) RunOnce(ctx context.Context) error {
	operations, err := w.Store.ListAcceptedOperations(ctx, w.batchSize())
	if err != nil {
		return err
	}
	for _, op := range operations {
		leased, err := w.Store.LeaseOperation(ctx, op.ID, w.owner(), w.leaseTTL())
		if err != nil {
			return err
		}
		if !leased {
			continue
		}
		if _, err := w.Orchestrator.Apply(ctx, op.ID); err != nil {
			if recordErr := w.Store.RecordOperationFailure(ctx, op.ID, err); recordErr != nil {
				return recordErr
			}
		}
	}
	return nil
}

func (w Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval())
	defer ticker.Stop()
	for {
		if err := w.RunOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w Worker) owner() string {
	if w.Owner == "" {
		return "fabric-worker"
	}
	return w.Owner
}

func (w Worker) batchSize() int {
	if w.BatchSize <= 0 {
		return 10
	}
	return w.BatchSize
}

func (w Worker) leaseTTL() time.Duration {
	if w.LeaseTTL <= 0 {
		return time.Minute
	}
	return w.LeaseTTL
}

func (w Worker) interval() time.Duration {
	if w.Interval <= 0 {
		return 5 * time.Second
	}
	return w.Interval
}
