# Migrations

The first Fabric schema is embedded into the API binary from `apps/fabric-api/internal/postgres/schema.sql` through `apps/fabric-api/internal/postgres/migrations.go`.

Run `Store.Migrate` against the target PostgreSQL database before serving mutating Fabric API traffic. The migration currently applies the embedded schema with idempotent `CREATE TABLE IF NOT EXISTS` statements.

The Kubernetes manifest expects `DATABASE_URL` in the `opl-fabric-api-secrets` Secret under the `DATABASE_URL` key:

```bash
kubectl create secret generic opl-fabric-api-secrets \
  --from-literal=DATABASE_URL='postgres://user:password@postgres:5432/opl_fabric?sslmode=disable' \
  --from-literal=OPL_OPERATOR_TOKEN='replace-with-operator-token' \
  --from-literal=TENCENT_MUTATION_SECRET_ID='replace-with-tencent-secret-id' \
  --from-literal=TENCENT_MUTATION_SECRET_KEY='replace-with-tencent-secret-key'
```
