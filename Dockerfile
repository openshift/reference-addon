FROM registry.access.redhat.com/ubi9/go-toolset:1.17.12-8 AS builder

WORKDIR /workspace

# prevents flakes with 'go mod tidy'
USER root

# main cache optimization
COPY go.mod .
COPY go.sum .
RUN go mod download

# tools cache optimization
COPY tools/ tools/
RUN cd tools && go mod download

# build manager binary
COPY . .
RUN ./mage build:manager

USER default

#9.0.0-21
FROM registry.access.redhat.com/ubi9-micro@sha256:2c97b1e127c01e33d9aee7da124a55a51ad8cdc634cf674ea3f6f69995ea3b65

COPY --from=builder /workspace/bin/linux_amd64/reference-addon-manager .

USER 65532:65532

ENTRYPOINT [ "/reference-addon-manager" ]
