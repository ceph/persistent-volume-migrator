# FROM golang:latest as builder
FROM docker.io/ceph/ceph:v16 as builder
WORKDIR "/go/src/github.com/local-migration/migrate-to-ceph-csi"
ARG GOROOT=/usr/local/go
ARG GO_ARCH="amd64"
ARG GOLANG_VERSION="1.16.4"

RUN mkdir -p ${GOROOT} && \
    curl https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-${GO_ARCH}.tar.gz | tar xzf - -C ${GOROOT} --strip-components=1

RUN ${GOROOT}/bin/go version && ${GOROOT}/bin/go env

RUN dnf -y install \
	librados-devel librbd-devel \
	/usr/bin/cc \
	make \
	git \
    && true

ENV GOROOT=${GOROOT} \
    GOPATH=/go \
    CGO_ENABLED=1 \
    GIT_COMMIT="${GIT_COMMIT}" \
    # ENV_CSI_IMAGE_VERSION="${CSI_IMAGE_VERSION}" \
    # ENV_CSI_IMAGE_NAME="${CSI_IMAGE_NAME}" \
    PATH="${GOROOT}/bin:${GOPATH}/bin:${PATH}"

COPY go.mod go.sum ./
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

######## Start a new stage from scratch #######
FROM docker.io/ceph/ceph:v16

# RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/src/github.com/local-migration/migrate-to-ceph-csi/main /usr/local/bin/main

# # Expose port 8080 to the outside world
# EXPOSE 8080


# verify that all dynamically linked libraries are available
RUN [ $(ldd /usr/local/bin/main | grep -c '=> not found') = '0' ]

ENTRYPOINT ["/usr/local/bin/main"]

# Command to run the executable
CMD ["./main"]