# Blink Labs does not publish a 1.26.5 image; use the patched official image.
FROM golang:1.26.5 AS build

WORKDIR /code
COPY . .
RUN make build

FROM cgr.dev/chainguard/glibc-dynamic AS ouroboros-mock
COPY --from=build /code/ouroboros-mock /bin/
ENTRYPOINT ["ouroboros-mock"]
