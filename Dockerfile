FROM golang:1.23 AS build-image
LABEL authors="bixority"

WORKDIR /build
ENV CGO_ENABLED=0

COPY ./ /build/

RUN make

FROM scratch
LABEL authors="bixority"

COPY --from=build-image /build/bin/pg_maintenance /build/LICENSE /

CMD ["/pg_maintenance"]
