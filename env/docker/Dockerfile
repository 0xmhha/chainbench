# A virtual server for chainbench's remote-path verification: plain ubuntu
# with sshd, reached by key only. The chain binary is NOT baked in — a server
# starts empty and provision puts material on it, the same as a real host.
FROM ubuntu:24.04

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        openssh-server \
    && rm -rf /var/lib/apt/lists/*

# Key-only login. The authorized key is mounted at runtime, not baked in, so
# rebuilding the image never invalidates the generated keypair.
RUN mkdir -p /run/sshd \
    && printf '%s\n' \
        'PermitRootLogin prohibit-password' \
        'PasswordAuthentication no' \
        'KbdInteractiveAuthentication no' \
        > /etc/ssh/sshd_config.d/chainbench.conf

EXPOSE 22
