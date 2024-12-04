FROM golang:1.23.2 AS build-image
LABEL authors="bixority"

ARG upx_version=4.2.4
ARG GOPROXY
ARG TARGETARCH=${TARGETARCH:-amd64}

WORKDIR /build
ENV CGO_ENABLED=0

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN apt-get update && apt-get install -y --no-install-recommends xz-utils && \
  curl -Ls https://github.com/upx/upx/releases/download/v${upx_version}/upx-${upx_version}-${TARGETARCH}_linux.tar.xz -o - | tar xvJf - -C /tmp && \
  cp /tmp/upx-${upx_version}-${TARGETARCH}_linux/upx /usr/local/bin/ && \
  chmod +x /usr/local/bin/upx && \
  apt-get remove -y xz-utils && \
  rm -rf /var/lib/apt/lists/* && \
  make

COPY ./ /build/

FROM scratch
LABEL authors="bixority"

COPY --from=build-image /build/bin/pg_maintenance /build/LICENSE /

CMD ["/pg_maintenance"]
