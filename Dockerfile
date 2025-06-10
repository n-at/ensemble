FROM golang:1.24 AS builder

ENV DEBIAN_FRONTEND=noninteractive

ADD . /build

RUN apt-get update &&\
    apt-get install -y git curl gpg &&\
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor >> /nodesource-key.gpg &&\
    echo "deb [signed-by=/nodesource-key.gpg] https://deb.nodesource.com/node_20.x bookworm main" >> /etc/apt/sources.list.d/nodesource.list &&\
    echo "deb-src [signed-by=/nodesource-key.gpg] https://deb.nodesource.com/node_20.x bookworm main" >> /etc/apt/sources.list.d/nodesource.list &&\
    apt-get install -y nodejs npm &&\
    \
    cd /build &&\
    CGO_ENABLED=0 GOOS=linux go build -a -o app . &&\
    cd assets &&\
    npm install

###############################################################################

FROM ubuntu:22.04

COPY --from=builder /build/assets /app/assets/
COPY --from=builder /build/templates /app/templates/
COPY --from=builder /build/ssh_add_key.sh /app/
COPY --from=builder /build/app /app/

ADD docker/run.sh /app/

ENV DEBIAN_FRONTEND=noninteractive
ENV SSH_AUTH_SOCK=/app/ssh-agent.sock

RUN apt-get update &&\
    apt-get install -y git openssh-client sshpass gnupg2 ca-certificates &&\
    echo "deb http://ppa.launchpad.net/ansible/ansible/ubuntu jammy main" >> /etc/apt/sources.list.d/ansible.list &&\
    apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 93C4A3FD7BB9C367 && \
    apt-get update &&\
    apt-get install -y ansible && \
    useradd --home-dir /home/ensemble --create-home --user-group --system ensemble &&\
    chmod 0777 /app &&\
    chown -R ensemble:ensemble /app

USER ensemble

RUN ansible-galaxy collection install ansible.posix

WORKDIR /app

EXPOSE 3000

VOLUME ["/app/data", "/app/keys"]

CMD ["./run.sh"]
