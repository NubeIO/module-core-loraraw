# Use Ubuntu 20.04 as the base image
FROM ubuntu:20.04 AS builder

# Install Go and necessary dependencies for cross-compilation
RUN apt-get update
RUN apt-get install -y \
    gcc \
    gcc-x86-64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    g++-arm-linux-gnueabihf \
    wget

# Set the working directory
WORKDIR /app

# Copy the Go project files
COPY . .

# Define build arguments
ARG VERSION
ARG COMMIT
ARG BUILD_DATE

RUN wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
        rm go1.21.13.linux-amd64.tar.gz

# Set Go environment variables
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go

# Run Go mod tidy and build for the default platform (amd64)
RUN go mod tidy
RUN env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
    CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++ \
    go build -o module-core-loraraw-amd64

# Build for ARMv7 architecture
RUN env GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 \
       CC=arm-linux-gnueabihf-gcc CXX=arm-linux-gnueabihf-g++ \
       go build -o module-core-loraraw-armv7