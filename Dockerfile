FROM ghcr.io/blinklabs-io/go:1.21.6-1 AS build

WORKDIR /code
COPY . .
RUN make build

FROM cgr.dev/chainguard/glibc-dynamic AS ouroboros-mock
COPY --from=build /code/ouroboros-mock /bin/
ENTRYPOINT ["ouroboros-mock"]
