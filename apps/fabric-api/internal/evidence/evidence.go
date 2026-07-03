package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrLedgerOperationIDRequired  = errors.New("ledger_operation_id_required")
	ErrLedgerPhaseRequired        = errors.New("ledger_phase_required")
	ErrLedgerStatusRequired       = errors.New("ledger_status_required")
	ErrLedgerSummaryRequired      = errors.New("ledger_summary_required")
	ErrLedgerActionRequired       = errors.New("ledger_entry_requires_action")
	ErrLedgerActionIDRequired     = errors.New("ledger_action_id_required")
	ErrLedgerActionKindRequired   = errors.New("ledger_action_kind_required")
	ErrLedgerActionTargetRequired = errors.New("ledger_action_target_required")
	ErrLedgerActionResultRequired = errors.New("ledger_action_result_required")
	ErrLedgerActionSHAInvalid     = errors.New("ledger_action_sha_invalid")
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
	if entry.OperationID == "" {
		return ErrLedgerOperationIDRequired
	}
	if entry.Phase == "" {
		return ErrLedgerPhaseRequired
	}
	if entry.Status == "" {
		return ErrLedgerStatusRequired
	}
	if entry.Summary == "" {
		return ErrLedgerSummaryRequired
	}
	if len(entry.Actions) == 0 {
		return ErrLedgerActionRequired
	}
	for _, action := range entry.Actions {
		if action.ActionID == "" {
			return ErrLedgerActionIDRequired
		}
		if action.ActionKind == "" {
			return ErrLedgerActionKindRequired
		}
		if action.TargetRef == "" {
			return ErrLedgerActionTargetRequired
		}
		if action.Result == "" {
			return ErrLedgerActionResultRequired
		}
		if !isSHA256Hex(action.SHA256) {
			return ErrLedgerActionSHAInvalid
		}
	}
	return nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
