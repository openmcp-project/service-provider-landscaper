# NOTE: This Dockerfile is used by the pipeline, but not for the 'make image' command, which uses the Dockerfile template in hack/common instead.
# Use distroless as minimal base image to package the component binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
WORKDIR /
COPY bin/$COMPONENT-$TARGETOS.$TARGETARCH /$COMPONENT
USER nonroot:nonroot

ENTRYPOINT ["/$COMPONENT"]
