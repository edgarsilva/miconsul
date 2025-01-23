FROM golang:1.23

# Install needed dependencies
RUN apt-get update && \
	apt-get install -y unzip --no-install-recommends && \
	rm -rf /var/lib/apt/lists/* tar curl wget

RUN groupadd -g 1000 miconsul && useradd -u 1000 -m -g miconsul miconsul
USER miconsul
WORKDIR /app

# USER miconsul
RUN mkdir /app/store

# Install bun (for TailwindCSS)
RUN bash -o pipefail -c "curl -fsSL https://bun.sh/install | bash"

# Set GOPATH to a directory where the app user has permission to write
ENV GOPATH=/home/miconsul/go

# Install templ (Go HTML template generator) and goose for DB migrations
RUN go install github.com/a-h/templ/cmd/templ@v0.3.819 && \
	go install github.com/pressly/goose/v3/cmd/goose@v3.20

# Build TailwindCSS plugins and the Go application
COPY package*.json ./
RUN ~/.bun/bin/bun install

# Download Go modules
COPY --chown=miconsul:miconsul go.mod go.sum ./
RUN go mod download all && go mod verify

# Copy app source code to /app inside container
COPY --chown=miconsul:miconsul . .

RUN make build

# Start the application
CMD ["make", "start"]
