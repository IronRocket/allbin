FROM golang:1.24.2-bookworm AS builder


WORKDIR /app

COPY . .
RUN go mod download


RUN go build -v -o server


FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/server /app/server
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/.env ./.env
COPY --from=builder /app/assets ./assets

RUN useradd -m nonroot
RUN chown -R nonroot:nonroot /app && chmod -R o= /app
USER nonroot

EXPOSE 5000

# Run the web service on container startup.
CMD ["/app/server"]