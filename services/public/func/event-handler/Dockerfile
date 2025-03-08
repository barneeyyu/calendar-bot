FROM golang:1.22 as builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -tags lambda.norpc \
    -o bootstrap ./services/public/func/event-handler

FROM public.ecr.aws/lambda/provided:al2-arm64

COPY --from=builder /app/bootstrap /var/runtime/bootstrap
RUN chmod +x /var/runtime/bootstrap