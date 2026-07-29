#!/usr/bin/env bash
set -Eeuo pipefail

PATH='/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'

readonly compose_file='/srv/fribench/apps/internal/fribench-monitor/current/compose.yaml'
readonly secret_file='/etc/fribench/fribench-monitor/password'
readonly secret_group='fribench-secrets'
readonly username='monitor'

backup_file=''
password_changed=0

die() {
  echo "ERROR: $*" >&2
  exit 1
}

rollback_password() {
  local exit_code=$?
  trap - ERR

  if [ "$password_changed" -eq 1 ] && [ -f "$backup_file" ]; then
    echo 'Password rotation failed; restoring the previous credential.' >&2
    /usr/bin/mv -f -- "$backup_file" "$secret_file"
    /usr/bin/docker compose -f "$compose_file" restart >/dev/null || true
  fi
  exit "$exit_code"
}

read_new_password() {
  local confirmation
  local password

  [ -r /dev/tty ] || die 'an interactive terminal is required'
  IFS= read -r -s -p 'New monitor password (20+ characters): ' password </dev/tty
  /usr/bin/printf '\n' >/dev/tty
  IFS= read -r -s -p 'Confirm new monitor password: ' confirmation </dev/tty
  /usr/bin/printf '\n' >/dev/tty

  [ "${#password}" -ge 20 ] || die 'password must contain at least 20 characters'
  [ "$password" = "$confirmation" ] || die 'password confirmation does not match'
  unset confirmation
  /usr/bin/printf '%s' "$password"
}

main() {
  local auth_header
  local authorized_status
  local new_password
  local status
  local temp_file

  [ "$(id -u)" -eq 0 ] || die 'run this script through sudo as root'
  [ -f "$compose_file" ] || die "missing Compose file: ${compose_file}"
  [ -s "$secret_file" ] || die "missing password file: ${secret_file}"
  /usr/bin/getent group "$secret_group" >/dev/null ||
    die "missing host group: ${secret_group}"

  new_password="$(read_new_password)"
  temp_file="${secret_file}.new.$$"
  backup_file="${secret_file}.previous.$$"

  /usr/bin/printf '%s\n' "$new_password" >"$temp_file"
  /usr/bin/chown root:"$secret_group" "$temp_file"
  /usr/bin/chmod 0640 "$temp_file"
  /usr/bin/cp -p -- "$secret_file" "$backup_file"

  trap rollback_password ERR
  /usr/bin/mv -f -- "$temp_file" "$secret_file"
  password_changed=1
  /usr/bin/docker compose -f "$compose_file" restart >/dev/null

  for _ in $(/usr/bin/seq 1 30); do
    if /usr/bin/curl -fsS --max-time 3 \
      http://127.0.0.1:26001/healthz >/dev/null; then
      break
    fi
    /usr/bin/sleep 1
  done
  /usr/bin/curl -fsS --max-time 3 \
    http://127.0.0.1:26001/healthz >/dev/null

  status="$(
    /usr/bin/curl -sS --max-time 3 -o /dev/null -w '%{http_code}' \
      http://127.0.0.1:26001/
  )"
  [ "$status" = '401' ] || return 1

  auth_header="$(
    /usr/bin/printf '%s:%s' "$username" "$new_password" |
      /usr/bin/base64 -w 0
  )"
  authorized_status="$(
    /usr/bin/printf 'header = "Authorization: Basic %s"\n' "$auth_header" |
      /usr/bin/curl --config - -sS --max-time 3 -o /dev/null -w '%{http_code}' \
        http://127.0.0.1:26001/
  )"
  unset auth_header new_password
  [ "$authorized_status" = '200' ] || return 1

  password_changed=0
  /usr/bin/rm -f -- "$backup_file"
  trap - ERR
  echo 'Fribench Monitor password rotated and verified.'
}

main "$@"
