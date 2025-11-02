FROM golang:1.25.3-trixie AS builder

WORKDIR /service

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /network-discovery-service


FROM gcr.io/distroless/base-debian13 AS service

WORKDIR /
COPY --from=builder /network-discovery-service /network-discovery-service

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/network-discovery-service"]