# 1.17.12-8
FROM registry.access.redhat.com/ubi9/go-toolset@sha256:fc7503a79725f82b72d337fdd694a58609c73b5a85eeb92f1771962526514e5d AS builder

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

# 9.0.0-1644
FROM registry.access.redhat.com/ubi9-minimal@sha256:a9df0929861456151442affa821caa685d5faeeee7ae799125fe730fe1c5dd0c

RUN microdnf install shadow-utils -y \
	&& groupadd --gid 1000 noroot \
	&& adduser \
		--no-create-home \
		--no-user-group \
		--uid 1000 \
		--gid 1000 \
		noroot

COPY --from=builder /opt/app-root/src/bin/linux_amd64/reference-addon-manager .
RUN chown noroot:noroot /reference-addon-manager

USER "noroot"

ENTRYPOINT [ "/reference-addon-manager" ]
