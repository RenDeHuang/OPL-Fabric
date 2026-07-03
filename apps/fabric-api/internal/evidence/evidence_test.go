package evidence

import "testing"

func TestDigestManifestIsStable(t *testing.T) {
	input := []byte(`{"kind":"Deployment","metadata":{"name":"opl-ws"}}`)
	first := Digest(input)
	second := Digest(input)
	if first != second {
		t.Fatalf("digest changed: %s != %s", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("digest length = %d", len(first))
	}
	if got, want := Digest(nil), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"; got != want {
		t.Fatalf("empty digest = %s, want %s", got, want)
	}
	if got, want := Digest([]byte("manifest")), "05b3abf2579a5eb66403cd78be557fd860633a1fe2103c7642030defe32c657f"; got != want {
		t.Fatalf("manifest digest = %s, want %s", got, want)
	}
}

func TestLifecycleLedgerEntry(t *testing.T) {
	entry := LedgerEntry{
		OperationID: "op-1",
		Phase:       "apply",
		Status:      "succeeded",
		Summary:     "applied deployment",
		Actions: []LedgerAction{
			{
				ActionID:   "act-1",
				ActionKind: "apply",
				TargetRef:  "deployment/opl-ws",
				Result:     "created",
				SHA256:     Digest([]byte("manifest")),
			},
		},
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("entry should validate: %v", err)
	}
}

func TestLifecycleLedgerEntryValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		entry LedgerEntry
		want  error
	}{
		{
			name:  "missing_operation_id",
			entry: LedgerEntry{Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{validAction()}},
			want:  ErrLedgerOperationIDRequired,
		},
		{
			name:  "missing_action",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary"},
			want:  ErrLedgerActionRequired,
		},
		{
			name:  "missing_action_target",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{{ActionID: "act-1", ActionKind: "apply", Result: "created", SHA256: Digest([]byte("manifest"))}}},
			want:  ErrLedgerActionTargetRequired,
		},
		{
			name:  "invalid_sha",
			entry: LedgerEntry{OperationID: "op-1", Phase: "apply", Status: "succeeded", Summary: "summary", Actions: []LedgerAction{{ActionID: "act-1", ActionKind: "apply", TargetRef: "deployment/opl-ws", Result: "created", SHA256: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}}},
			want:  ErrLedgerActionSHAInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.entry.Validate(); err != tt.want {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func validAction() LedgerAction {
	return LedgerAction{
		ActionID:   "act-1",
		ActionKind: "apply",
		TargetRef:  "deployment/opl-ws",
		Result:     "created",
		SHA256:     Digest([]byte("manifest")),
	}
}
