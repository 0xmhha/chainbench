# A virtual server for chainbench's remote-path verification: plain ubuntu
# with sshd, reached the way the real fleet is reached — a user id and a
# password, with sudo that asks for that password. The chain binary is NOT
# baked in; a server starts empty and provision puts material on it.
FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        openssh-server sudo \
    && rm -rf /var/lib/apt/lists/*

# The login mirrors production shape: a named user with a password, in the
# sudo group, and sudo REQUIRES the password (ubuntu's default) — that is the
# path the harness has to prove, so NOPASSWD would defeat the purpose. The
# password is a fixed dev-only value on a loopback-published container.
RUN useradd -m -s /bin/bash chainbench \
    && echo 'chainbench:chainbench' | chpasswd \
    && usermod -aG sudo chainbench

# Password login only, no root: the same surface a hardened real host offers.
RUN mkdir -p /run/sshd \
    && printf '%s\n' \
        'PermitRootLogin no' \
        'PasswordAuthentication yes' \
        'PubkeyAuthentication no' \
        'KbdInteractiveAuthentication no' \
        > /etc/ssh/sshd_config.d/chainbench.conf

EXPOSE 22
