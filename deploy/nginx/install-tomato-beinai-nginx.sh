#!/usr/bin/env bash
set -euo pipefail

DOMAIN="tomato.beinai.cc"
APP_UPSTREAM="${APP_UPSTREAM:-127.0.0.1:18089}"
NGINX_CONF_DIR="${NGINX_CONF_DIR:-/etc/nginx/conf.d}"
SSL_DIR="${SSL_DIR:-/etc/nginx/ssl/${DOMAIN}}"
CONF_PATH="${NGINX_CONF_DIR}/${DOMAIN}.conf"
BACKUP_DIR="/root/nginx-backup-${DOMAIN}-$(date +%Y%m%d-%H%M%S)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LOCAL_SSL_DIR="${REPO_ROOT}/deploy/ssl/${DOMAIN}"
LOCAL_CONF="${SCRIPT_DIR}/${DOMAIN}.conf"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root: sudo bash $0"
  exit 1
fi

if [[ ! -f "${LOCAL_CONF}" ]]; then
  echo "Missing Nginx config: ${LOCAL_CONF}"
  exit 1
fi

if [[ ! -f "${LOCAL_SSL_DIR}/${DOMAIN}_bundle.crt" || ! -f "${LOCAL_SSL_DIR}/${DOMAIN}.key" ]]; then
  echo "Missing certificate files in ${LOCAL_SSL_DIR}"
  exit 1
fi

mkdir -p "${BACKUP_DIR}" "${NGINX_CONF_DIR}" "${SSL_DIR}"

if [[ -f "${CONF_PATH}" ]]; then
  cp -a "${CONF_PATH}" "${BACKUP_DIR}/"
fi

cp -a "${LOCAL_SSL_DIR}/${DOMAIN}_bundle.crt" "${SSL_DIR}/"
cp -a "${LOCAL_SSL_DIR}/${DOMAIN}.key" "${SSL_DIR}/"
chmod 644 "${SSL_DIR}/${DOMAIN}_bundle.crt"
chmod 600 "${SSL_DIR}/${DOMAIN}.key"
chown -R root:root "${SSL_DIR}"

sed "s/server 127\\.0\\.0\\.1:18089;/server ${APP_UPSTREAM};/" "${LOCAL_CONF}" > "${CONF_PATH}"

if ! nginx -t; then
  echo "nginx -t failed, rolling back only ${DOMAIN} config."
  rm -f "${CONF_PATH}"
  if [[ -f "${BACKUP_DIR}/${DOMAIN}.conf" ]]; then
    cp -a "${BACKUP_DIR}/${DOMAIN}.conf" "${CONF_PATH}"
  fi
  nginx -t || true
  exit 1
fi

systemctl reload nginx

echo "Installed ${DOMAIN} without modifying other site configs."
echo "Backup directory: ${BACKUP_DIR}"
nginx -T 2>/dev/null | grep -nE "server_name ${DOMAIN}|listen 443|ssl_certificate .*${DOMAIN}" || true
