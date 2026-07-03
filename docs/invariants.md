# Invariants

- Fabric owns only Fabric resource truth and provider execution evidence.
- Compute destruction never destroys storage.
- Storage destruction requires explicit confirmation or a recorded human gate.
- Backup deletion never deletes source storage, restored storage, or compute.
- Restore creates new storage and never overwrites an existing PVC.
- Normal provider execution uses Go client-go, not kubectl subprocesses.
- Machine-readable truth lives in contracts.
