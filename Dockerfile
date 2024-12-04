FROM golang:1.23 AS build-image
LABEL authors="bixority"

WORKDIR /build
ENV CGO_ENABLED=0

RUN apt update && apt install -y upx-ucl && make

COPY ./ /build/

FROM scratch
LABEL authors="bixority"

COPY --from=build-image /build/bin/pg_maintenance /build/LICENSE /

CMD ["/pg_maintenance"]
