FROM scratch
USER 65532:65532
ENTRYPOINT ["/rbac-operator"]
COPY rbac-operator /