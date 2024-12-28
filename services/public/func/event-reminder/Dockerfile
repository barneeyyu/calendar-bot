FROM golang:1.21 as builder

WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=arm64 go build -o bootstrap ./services/public/func/event-reminder

FROM public.ecr.aws/lambda/provided:al2-arm64

COPY --from=builder /app/bootstrap /var/runtime/bootstrap
RUN chmod +x /var/runtime/bootstrap