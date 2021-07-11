FROM golang as builder

WORKDIR /app
COPY ./ /app

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' cmd/ctff-server/main.go

FROM scratch
COPY --from=builder /app/main /main
ENTRYPOINT [ "/main" ]
