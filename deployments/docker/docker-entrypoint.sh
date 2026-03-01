#!/bin/sh
set -e

# Fix DNS resolution for podman - add short name aliases to /etc/hosts
# Podman uses .dns.podman suffix but nginx config uses short names

add_host_if_missing() {
    hostname="$1"
    if ! grep -q "^.*\s${hostname}\s" /etc/hosts; then
        # Try to resolve using podman DNS
        ip=$(getent hosts "${hostname}.dns.podman" 2>/dev/null | awk '{print $1}')
        if [ -n "$ip" ]; then
            echo "$ip $hostname" >> /etc/hosts
            echo "Added $hostname -> $ip to /etc/hosts"
        else
            echo "Warning: Could not resolve $hostname.dns.podman"
        fi
    fi
}

# Services that nginx needs to resolve
add_host_if_missing "auth-service"
add_host_if_missing "registry-service"
add_host_if_missing "job-service"
add_host_if_missing "storage-service"
add_host_if_missing "notification-service"

# Execute the main command
exec "$@"
