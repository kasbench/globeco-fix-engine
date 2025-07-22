# syntax=docker/dockerfile:1.4

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /src/cmd/fix-engine

# Build for target platform with explicit architecture
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /out/globeco-fix-engine

FROM --platform=$TARGETPLATFORM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/globeco-fix-engine /globeco-fix-engine
COPY --from=builder /src/migrations /migrations
EXPOSE 8085
USER nonroot
ENTRYPOINT ["/globeco-fix-engine"] 