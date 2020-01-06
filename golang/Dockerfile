FROM golang:1.13-alpine AS builder

# RUN apk add --no-cache make

COPY .  /go/src/github.com/x0rxkov/arXiv2git
WORKDIR /go/src/github.com/x0rxkov/arXiv2git

RUN cd /go/src/github.com/x0rxkov/arXiv2git \
 && go build -v

FROM alpine:3.11 AS runtime

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/v0.18.0/tini-muslc-amd64 /usr/local/sbin/tini

# Install runtime dependencies & create runtime user
RUN apk --no-cache --no-progress add ca-certificates \
 && chmod +x /usr/local/sbin/tini && mkdir -p /opt \
 && adduser -D a2g -h /opt/arXiv2git -s /bin/sh \
 && su a2g -c 'cd /opt/arXiv2git; mkdir -p bin config data'

# Switch to user context
USER a2g
WORKDIR /opt/arXiv2git

# Copy git2etcd binary to /opt/git2etcd/bin
COPY --from=builder /go/src/github.com/blippar/arXiv2git/arXiv2git /opt/arXiv2git/bin/arXiv2git
# COPY config.example.json /opt/arXiv2git/config/config.json
ENV PATH $PATH:/opt/arXiv2git/bin

# Container configuration
EXPOSE 4242
VOLUME ["/opt/arXiv2git/data"]
ENTRYPOINT ["tini", "-g", "--"]
CMD ["/opt/arXiv2git/bin/arXiv2git"]
# CMD ["/opt/arXiv2git/bin/arXiv2git", "-conf_dir=/opt/arXiv2git/config"]
