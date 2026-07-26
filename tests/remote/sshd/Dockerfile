# A minimal Linux SSH server that stands in for a remote host so the SSH
# RemoteDriver (pkg/core/driver) can be exercised end to end locally. It ships a
# fake node binary (`init` exits 0; any other command sleeps) so Provision /
# Launch / Stop work without a real chain binary.
#
# Build + run: tests/remote/sshd/run.sh
FROM debian:stable-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends openssh-server \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -p /run/sshd

# A non-root user with a known password (test fixture only).
RUN useradd -m -s /bin/bash chainbench \
 && echo 'chainbench:chainbench' | chpasswd

# Allow password auth (default on Debian, pinned here for clarity).
RUN sed -ri 's/^#?PasswordAuthentication .*/PasswordAuthentication yes/' /etc/ssh/sshd_config \
 && sed -ri 's/^#?PermitRootLogin .*/PermitRootLogin no/' /etc/ssh/sshd_config

# Fake node binary the RemoteDriver launches: `init` succeeds, anything else
# runs indefinitely (so Launch returns a live PID that Stop can kill).
RUN printf '#!/bin/sh\ncase "$1" in\n  init) exit 0 ;;\n  *) exec sleep 3600 ;;\nesac\n' \
      > /usr/local/bin/fakenode \
 && chmod +x /usr/local/bin/fakenode

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
