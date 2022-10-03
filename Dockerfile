# 1.17.12-8
FROM registry.access.redhat.com/ubi9/go-toolset@sha256:fc7503a79725f82b72d337fdd694a58609c73b5a85eeb92f1771962526514e5d AS builder

# main cache optimization
COPY go.mod .
COPY go.sum .
RUN go mod download

# build manager binary
COPY . .
RUN ./mage build:manager

# 9.0.0-21
FROM registry.access.redhat.com/ubi9-micro@sha256:421fa580d588ca9f25b8b397b54b75db55ea2995a4d2320ce11b7223d433948a

COPY --from=builder /opt/app-root/src/bin/linux_amd64/reference-addon-manager .

ENTRYPOINT [ "/reference-addon-manager" ]
