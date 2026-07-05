FROM golang:1.23-bookworm AS builder

WORKDIR /src/apps/fabric-api

COPY apps/fabric-api/go.mod apps/fabric-api/go.sum ./
RUN go mod download

COPY apps/fabric-api/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/opl-fabric-api ./cmd/fabric-api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/opl-fabric-api /app/opl-fabric-api

USER 65532:65532
EXPOSE 8787

ENTRYPOINT ["/app/opl-fabric-api"]
