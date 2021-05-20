FROM scratch

WORKDIR /
COPY passwd /etc/passwd
COPY reference-addon-manager /

USER "noroot"

ENTRYPOINT ["/reference-addon-manager"]
