ARG BUILDPATH='/build/'

FROM golang:1.16 as builder

ARG BUILDPATH

WORKDIR $BUILDPATH

# Copy code in && fetch dependencies
COPY . ./

RUN go mod download

# Dots API - Assumes AMD64 arch
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o button

# Create final imag
FROM alpine:3.14

ARG BUILDPATH

WORKDIR $BUILDPATH

COPY --from=builder $BUILDPATH .

CMD ["./button"]