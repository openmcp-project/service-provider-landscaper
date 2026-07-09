# NOTE: This Dockerfile is used by the pipeline, but not for the 'make image' command, which uses the Dockerfile template in hack/common instead.
# Use distroless as minimal base image to package the component binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478
ARG TARGETOS
ARG TARGETARCH
ARG COMPONENT
WORKDIR /
COPY bin/$COMPONENT-$TARGETOS.$TARGETARCH /$COMPONENT
USER nonroot:nonroot

ENTRYPOINT ["/$COMPONENT"]
