# Crowd-Volt fork build: layer our patched Go binary + frontend on top of the official
# OSS community image. This avoids re-running the upstream's complex multi-stage build
# (apk, ca-certs, etc.) and lets the fork-sync GitHub Action stay small + fast.
#
# Inputs (build context):
#   target/signoz-linux-amd64         — Go binary built from cmd/community with linux/amd64
#   target/web                        — frontend build output (frontend/build)
#
# Build:
#   cd /repo/root
#   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
#     -ldflags="-X github.com/SigNoz/signoz/pkg/version.variant=community ..." \
#     -o target/signoz-linux-amd64 ./cmd/community
#   cd frontend && yarn install --frozen-lockfile && yarn build && cd ..
#   mkdir -p target/web && cp -R frontend/build/. target/web/
#   docker build -f docker/fork.Dockerfile -t <registry>/signoz-trustedheader:<tag> target/
ARG UPSTREAM_IMAGE=signoz/signoz-community:v0.122.0
FROM ${UPSTREAM_IMAGE}

# The base image's ENTRYPOINT is `./signoz-community server`. Replace the binary in-place.
COPY signoz-linux-amd64 /root/signoz-community
COPY web                /etc/signoz/web
