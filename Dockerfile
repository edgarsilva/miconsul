FROM golang:1.23

# Install needed dependencies
RUN apt update && apt install -y unzip tar curl wget

# Install just utility
RUN curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

RUN groupadd miconsul && useradd -r -m -g miconsul miconsul
USER miconsul

# Set destination for COPY
WORKDIR /home/miconsul/app

# Install bun (for TailwindCSS)
RUN curl -fsSL https://bun.sh/install | bash

# Set GOPATH to a directory where the app user has permission to write
ENV GOPATH=/home/miconsul/go

# Install templ (Go HTML template generator)
RUN go install github.com/a-h/templ/cmd/templ@v0.2.778

# Install goose for DB migrations
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.20

# Build TailwindCSS plugins and the Go application
COPY --chown=miconsul:miconsul package*.json ./

RUN ~/.bun/bin/bun install

# Download Go modules
COPY --chown=miconsul:miconsul go.mod go.sum ./
RUN go mod download && go mod verify

# Copy app source code to /app inside container
COPY --chown=miconsul:miconsul . .

RUN just build

# Start the application
CMD ["just", "start"]
