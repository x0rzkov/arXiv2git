FROM golang:1.13-alpine AS builder

# RUN apk add --no-cache make

COPY .  /go/src/github.com/x0rxkov/arXiv2git
WORKDIR /go/src/github.com/x0rxkov/arXiv2git

RUN cd /go/src/github.com/x0rxkov/arXiv2git \
 && go install \
 && ls -lhS /go/bin

FROM alpine:3.11 AS runtime

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/v0.18.0/tini-muslc-amd64 /usr/local/sbin/tini

# Install runtime dependencies & create runtime user
RUN apk --no-cache --no-progress add ca-certificates \
 && chmod +x /usr/local/sbin/tini && mkdir -p /opt \
 && adduser -D a2g -h /opt/app -s /bin/sh \
 && su a2g -c 'cd /opt/app; mkdir -p bin config data'

# Switch to user context
USER a2g
WORKDIR /opt/app

# Copy git2etcd binary to /opt/git2etcd/bin
COPY --from=builder /go/bin/golang /opt/app/bin/arXiv2git
# COPY config.example.json /opt/arXiv2git/config/config.json
ENV PATH $PATH:/opt/app/bin

# Container configuration
EXPOSE 4242
VOLUME ["/opt/app/data"]
ENTRYPOINT ["tini", "-g", "--"]
CMD ["/opt/app/bin/arXiv2git"]
# CMD ["/opt/app/bin/arXiv2git", "-conf_dir=/opt/app/config"]
