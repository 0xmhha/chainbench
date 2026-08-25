# A virtual server for chainbench's remote-path verification: plain ubuntu
# with sshd, reached the way the real fleet is reached — a user id and a
# password, with sudo that asks for that password. The chain binary is NOT
# baked in; a server starts empty and provision puts material on it.
FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        openssh-server sudo \
    && rm -rf /var/lib/apt/lists/*

# No accounts are baked in: every login (including the harness's) comes from
# env/docker/accounts.env, injected at container start by setup-accounts.sh —
# same shape as the real fleet, where access is a provisioned dev account.
# sudo REQUIRES the password (ubuntu's default): that flow is what the
# harness has to prove, so NOPASSWD would defeat the purpose.

# Password login only, no root: the same surface a hardened real host offers.
RUN mkdir -p /run/sshd \
    && printf '%s\n' \
        'PermitRootLogin no' \
        'PasswordAuthentication yes' \
        'PubkeyAuthentication no' \
        'KbdInteractiveAuthentication no' \
        > /etc/ssh/sshd_config.d/chainbench.conf

EXPOSE 22
