package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

type LedgerEntry struct {
	OperationID string         `json:"operationId"`
	Phase       string         `json:"phase"`
	Status      string         `json:"status"`
	Summary     string         `json:"summary"`
	Actions     []LedgerAction `json:"actions"`
}

type LedgerAction struct {
	ActionID   string `json:"actionId"`
	ActionKind string `json:"actionKind"`
	TargetRef  string `json:"targetRef"`
	Result     string `json:"result"`
	SHA256     string `json:"sha256"`
}

func Digest(input []byte) string {
	sum := sha256.Sum256(input)
	return hex.EncodeToString(sum[:])
}

func (entry LedgerEntry) Validate() error {
	if entry.OperationID == "" || entry.Phase == "" || entry.Status == "" || entry.Summary == "" {
		return errors.New("ledger_entry_missing_required_field")
	}
	if len(entry.Actions) == 0 {
		return errors.New("ledger_entry_requires_action")
	}
	for _, action := range entry.Actions {
		if action.ActionID == "" || action.ActionKind == "" || action.TargetRef == "" || action.Result == "" || len(action.SHA256) != 64 {
			return errors.New("ledger_action_invalid")
		}
	}
	return nil
}
