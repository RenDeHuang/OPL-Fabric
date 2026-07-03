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
