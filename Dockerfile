FROM golang:1.22

# Set destination for COPY
WORKDIR /app

RUN apt-get update
RUN apt-get install -y unzip tar

RUN echo "🥐 Installing bun (for tailwindcss)"
RUN curl -fsSL https://bun.sh/install | bash

RUN echo "🌬️ Installing TailwindCSS plugins"
RUN ~/.bun/bin/bun add -D tailwindcss
RUN ~/.bun/bin/bun add -D daisyui@latest
RUN ~/.bun/bin/bun add -D @tailwindcss/typography

RUN echo "🛕 installing Templ"
RUN go install github.com/a-h/templ/cmd/templ@latest

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY ./ .

# Build
RUN make build

# Start
CMD ["make", "start"]

