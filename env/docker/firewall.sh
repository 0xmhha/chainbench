#!/bin/sh
# firewall.sh — the Wemix3.5 test servers' firewall, applied inside each
# container at start so the virtual servers refuse what the real ones refuse.
# Runs as root (before sshd); needs NET_ADMIN (compose grants it).
#
# Open (TCP): 10022 ssh · auth/http/ws bands sized by SLOTS (default 4, so
#             8501-8504 · 8601-8604 · 8701-8704) · 6060 metric · 3000 eth-stats ·
#             3001 grafana · 9100 node_exporter · 9090 prometheus ·
#             p2p from 30301 across SLOTS*3 ports · 1099 jmeter rmi · 5901 vnc ·
#             5044 logstash · 9200 elasticsearch
# Open (UDP): 30303 bootnode
# Everything else inbound: DROP.
set -eu

ipt() { iptables "$@"; }

ipt -F INPUT
ipt -P INPUT DROP
ipt -A INPUT -i lo -j ACCEPT
ipt -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
ipt -A INPUT -p icmp -j ACCEPT

for p in 10022 6060 3000 3001 9100 9090 1099 5901 5044 9200; do
    ipt -A INPUT -p tcp --dport "$p" -j ACCEPT
done
# The per-purpose bands, one port per slot. SLOTS comes from the same knob
# gen-env.sh uses to build the server set and publish the ports, because a
# firewall that opens four while the set declares six is a node that runs,
# answers on its own machine, and reads as dead from outside — which is exactly
# how a 5-node handoff lost its fifth node.
SLOTS="${SLOTS:-4}"
# p2p reserves two consecutive ports per node (p2p + etcd) and may step wider
# than one, so its band is sized from the step rather than the slot count.
P2P_SPAN="${P2P_SPAN:-$((SLOTS * 3))}"
for r in "8501:$((8501 + SLOTS - 1))" "8601:$((8601 + SLOTS - 1))" "8701:$((8701 + SLOTS - 1))" "30301:$((30301 + P2P_SPAN))"; do
    ipt -A INPUT -p tcp --dport "$r" -j ACCEPT
done
ipt -A INPUT -p udp --dport 30303 -j ACCEPT

echo "firewall: applied (default DROP, $(iptables -S INPUT | grep -c ACCEPT) accepts)"
