FROM golang:1.20-alpine AS builder
WORKDIR /avito-internship-2023
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY ./ ./
WORKDIR /avito-internship-2023/cmd/app
RUN CGO_ENABLED=0 go build -o /bin/app
WORKDIR /avito-internship-2023

FROM alpine:latest
RUN apk --update add ca-certificates
COPY --from=builder /bin/app /bin/app
COPY --from=builder /avito-internship-2023/deploy_user_segmenting/.env /deploy_user_segmenting/.env

CMD [ "/bin/app" ]