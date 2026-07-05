# Invariants

- Fabric owns only Fabric resource truth and provider execution evidence.
- Storage is persistent user-owned state; compute is rebuildable execution capacity.
- A ComputePool is shared by package or provider instance type.
- A ComputeAllocation is workspace-exclusive while active.
- A Workspace entry requires an attached storage volume.
- Compute destruction never destroys storage.
- Storage can be reattached after compute destruction.
- Storage destruction requires explicit confirmation or a recorded human gate.
- Normal provider execution uses Go client-go, not kubectl subprocesses.
- Normal Tencent Cloud capacity execution uses Tencent Cloud Go SDK, not tccli subprocesses.
- JavaScript provider runtime is not part of the long-term OPL Fabric runtime.
- Machine-readable truth lives in contracts.
