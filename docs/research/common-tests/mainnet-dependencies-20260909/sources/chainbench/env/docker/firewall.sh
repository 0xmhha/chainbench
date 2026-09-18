#!/bin/sh
# firewall.sh — the Wemix3.5 test servers' firewall, applied inside each
# container at start so the virtual servers refuse what the real ones refuse.
# Runs as root (before sshd); needs NET_ADMIN (compose grants it).
#
# Open (TCP): 10022 ssh · 8501-8504 wbft auth · 8601-8604 wbft http ·
#             8701-8704 wbft ws · 6060 metric · 3000 eth-stats ·
#             3001 grafana · 9100 node_exporter · 9090 prometheus ·
#             30301-30304 p2p · 1099 jmeter rmi · 5901 vnc ·
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
for r in 8501:8504 8601:8604 8701:8704 30301:30304; do
    ipt -A INPUT -p tcp --dport "$r" -j ACCEPT
done
ipt -A INPUT -p udp --dport 30303 -j ACCEPT

echo "firewall: applied (default DROP, $(iptables -S INPUT | grep -c ACCEPT) accepts)"
