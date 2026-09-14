FROM golang:1.26.5@sha256:705e964a93a2fd2e75c7d59bb7d781b57e30f12293ffde5175c69229e18fb678 AS build

WORKDIR /code
COPY . .
RUN make build

FROM cgr.dev/chainguard/glibc-dynamic AS ouroboros-mock
COPY --from=build /code/ouroboros-mock /bin/
ENTRYPOINT ["ouroboros-mock"]
