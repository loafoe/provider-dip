FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY _output/bin/linux_arm64/provider /provider
USER 65532:65532
ENTRYPOINT ["/provider"]
