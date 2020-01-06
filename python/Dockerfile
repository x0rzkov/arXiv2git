FROM python:3-alpine AS builder

WORKDIR /opt/app

# Copy requirements file
COPY requirements.txt /opt/app/requirements.txt

# Install Pipenv
RUN pip3 install --upgrade pip \
    && pip3 install pipenv

# Create a virtual environment and activate it
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH" \
	VIRTUAL_ENV="/opt/venv"

RUN pip3 install -r requirements.txt

FROM python:3-alpine AS runtime

MAINTAINER x0rxkov <x0rxkov@protonmail.com>

ARG VERSION
ARG BUILD
ARG NOW

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/v0.18.0/tini-muslc-amd64 /usr/local/sbin/tini

# Create runtime user
RUN mkdir -p /opt \
	&& adduser -D arxiv2git -h /opt/app -s /bin/sh \
 	&& su arxiv2git -c 'cd /opt/app; mkdir -p data conf' \
	&& chmod +x /usr/local/sbin/tini

COPY docker-entrypoint.sh /opt/app/docker-entrypoint.sh
RUN chmod +x /opt/app/docker-entrypoint.sh

# Switch to user context
USER arxiv2git
WORKDIR /opt/arx2git/data

# Copy the virtual environment from the previous image
COPY --from=builder /opt/venv /opt/venv

# Activate the virtual environment
ENV PATH="/opt/venv/bin:$PATH" \
	VIRTUAL_ENV="/opt/venv"

COPY 0.pull_repo_list.py /opt/app/0.pull_repo_list.py
COPY 1.fetch_README.py /opt/app/1.fetch_README.py
COPY 2.extract_arxiv_links.py /opt/app/2.extract_arxiv_links.py
COPY utils.py /opt/app/utils.py
COPY config.py /opt/app/config.py
# COPY docker-entrypoint.sh /opt/app/docker-entrypoint.sh
# RUN chmod +x /opt/app/docker-entrypoint.sh

LABEL name="arXiv2git" \
      version="$VERSION" \
      build="$BUILD" \
      architecture="x86_64" \
      build_date="$NOW" \
      vendor="thoppe" \
      maintainer="x0rzkov <x0rzkov@protonmail.com>" \
      url="https://github.com/x0rzkov/arXiv2git" \
      summary="Chrome extension that links arXiv papers to github repos" \
      description="Links the arXiv to github repos. Implemented as a chrome extension." \
      vcs-type="git" \
      vcs-url="https://github.com/x0rzkov/arXiv2git" \
      vcs-ref="$VERSION" \
      distribution-scope="public"

# ENTRYPOINT ["tini", "-g", "--"]
ENTRYPOINT ["/opt/app/docker-entrypoint.sh"]
