FROM ubuntu:latest as build-image
LABEL authors="kamikaze"

WORKDIR /build
COPY ./ /build/

FROM scratch:latest
LABEL authors="kamikaze"

COPY --from=build-image /bin/pg_maintenance /LICENSE /

CMD ["/pg_maintenance"]
