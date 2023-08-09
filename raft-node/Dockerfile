FROM golang:1.18 as build

WORKDIR /home

# Fetch dependencies
COPY go.* .
RUN go mod download

# Copy everything over except content in .dockerignore
COPY . .

RUN go build .

RUN chmod +x ./scalable-raft-demo

CMD ["./scalable-raft-demo"]
