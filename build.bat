@echo off

set MODULE_NAME=module-core-loraraw

docker build -t module-builder -f Dockerfile.module --build-arg MODULE_NAME=%MODULE_NAME% .
docker run -d --name module-builder module-builder:latest
docker container cp module-builder:/app/%MODULE_NAME% .
docker rm -f module-builder