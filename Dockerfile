ARG ALPINE_VERSION="3.17"
ARG GO_VERSION="1.19-alpine3.17"

##

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS binary_builder

ARG TARGETOS
ARG TARGETARCH

# Install the system dependencies.
RUN apk add --no-cache upx

# Install the project dependencies.
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

# Build and compress the binary for the target architecture and operating system.
COPY . .
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -v -ldflags="-w -s" -o dist/ . && \
    upx --best --lzma dist/ghcr-cleaning-action

##

FROM alpine:${ALPINE_VERSION}

# No USER instruction as a container GitHub action must be run as root.
# See https://docs.github.com/en/actions/creating-actions/dockerfile-support-for-github-actions#user

# Copy the static executable from the builder stage.
COPY --from=binary_builder /build/dist/ghcr-cleaning-action /usr/local/bin/

# Set the container entrypoint.
ENTRYPOINT ["/usr/local/bin/ghcr-cleaning-action"]
