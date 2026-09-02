#!/usr/bin/env bash
set -Eeuo pipefail

PATH='/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin'

readonly app_root='/srv/fribench/ops/services/fribench-monitor'
readonly ops_root='/srv/fribench/ops'
readonly current_link="${app_root}/current"
readonly secret_dir='/etc/fribench/fribench-monitor'
readonly secret_file="${secret_dir}/password"
readonly secret_group='fribench-secrets'
readonly username='monitor'

release_path="$(
  cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." &&
    pwd -P
)"
old_target=''
current_switched=0
password_generated=0

die() {
  echo "ERROR: $*" >&2
  exit 1
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    die 'run this rollout through sudo as root'
  fi
}

validate_inputs() {
  case "$release_path" in
    "${app_root}/releases/"*) ;;
    *) die "release must be staged below ${app_root}/releases" ;;
  esac

  [ -f "${release_path}/compose.yaml" ] ||
    die "missing release Compose file: ${release_path}/compose.yaml"
  [ -x "${ops_root}/scripts/fribench-direct-ports" ] ||
    die 'updated direct-port controller is not deployed'
  [ -f "${ops_root}/edge/nginx/sites-available/fribench-direct-ports.conf" ] ||
    die 'updated direct-port Nginx template is not deployed'
  /usr/bin/getent group "$secret_group" >/dev/null ||
    die "missing host group: ${secret_group}"

  /usr/bin/grep -Fq \
    'readonly -a public_ports=(21000 21001 21002 23001 23002 23003 23004 24001)' \
    "${ops_root}/scripts/fribench-direct-ports" ||
    die 'deployed direct-port controller does not include 21002'
  /usr/bin/grep -Fq 'listen 10.6.0.9:21002;' \
    "${ops_root}/edge/nginx/sites-available/fribench-direct-ports.conf" ||
    die 'deployed Nginx template does not include 21002'
}

prepare_secret() {
  local password
  local temp_file

  /usr/bin/install -d -o root -g "$secret_group" -m 0750 "$secret_dir"
  if [ -e "$secret_file" ]; then
    [ -f "$secret_file" ] || die "secret path is not a regular file: ${secret_file}"
    [ -s "$secret_file" ] || die "secret file is empty: ${secret_file}"
  else
    password="$(
      /usr/bin/openssl rand -base64 24 |
        /usr/bin/tr -d '\r\n'
    )"
    temp_file="${secret_dir}/.password.$$"
    /usr/bin/printf '%s\n' "$password" >"$temp_file"
    /usr/bin/chown root:"$secret_group" "$temp_file"
    /usr/bin/chmod 0640 "$temp_file"
    /usr/bin/mv -f -- "$temp_file" "$secret_file"
    password_generated=1
  fi

  [ "$(/usr/bin/stat -c '%U:%G:%a' "$secret_file")" = \
    "root:${secret_group}:640" ] ||
    die "secret file must be root:${secret_group} mode 0640"
}

install_ingress_assets() {
  /usr/bin/install -o root -g root -m 0755 \
    "${ops_root}/scripts/fribench-direct-ports" \
    /usr/local/sbin/fribench-direct-ports
  /usr/bin/install -d -o root -g root -m 0755 \
    /usr/local/share/fribench
  /usr/bin/install -o root -g root -m 0644 \
    "${ops_root}/edge/nginx/sites-available/fribench-direct-ports.conf" \
    /usr/local/share/fribench/direct-ports.nginx.conf
}

restore_previous_release() {
  local exit_code=$?
  local rollback_link
  trap - ERR

  if [ "$current_switched" -eq 1 ] && [ -n "$old_target" ]; then
    echo 'Monitor rollout failed; restoring the previous release.' >&2
    rollback_link="${app_root}/.current.rollback.$$"
    /usr/bin/ln -s "$old_target" "$rollback_link"
    /usr/bin/mv -Tf -- "$rollback_link" "$current_link"
    /usr/bin/docker compose -f "${old_target}/compose.yaml" up -d --build ||
      true
  fi
  exit "$exit_code"
}

activate_release() {
  local password
  local next_link
  local authorized_status_code
  local status_code

  old_target="$(/usr/bin/readlink -f "$current_link")"
  [ -n "$old_target" ] || die "cannot resolve current release: ${current_link}"

  /usr/bin/docker compose -f "${release_path}/compose.yaml" config >/dev/null
  /usr/bin/docker compose -f "${release_path}/compose.yaml" build

  trap restore_previous_release ERR
  next_link="${app_root}/.current.next.$$"
  /usr/bin/ln -s "$release_path" "$next_link"
  /usr/bin/mv -Tf -- "$next_link" "$current_link"
  current_switched=1

  /usr/bin/docker compose -f "${current_link}/compose.yaml" up -d

  for _ in $(/usr/bin/seq 1 30); do
    if /usr/bin/curl -fsS --max-time 3 \
      http://127.0.0.1:26001/healthz >/dev/null; then
      break
    fi
    /usr/bin/sleep 1
  done
  /usr/bin/curl -fsS --max-time 3 \
    http://127.0.0.1:26001/healthz >/dev/null

  status_code="$(
    /usr/bin/curl -sS --max-time 3 -o /dev/null -w '%{http_code}' \
      http://127.0.0.1:26001/
  )"
  if [ "$status_code" != '401' ]; then
    echo "ERROR: monitor dashboard must require authentication; received HTTP ${status_code}" >&2
    return 1
  fi

  password="$(/usr/bin/cat "$secret_file")"
  authorized_status_code="$(
    /usr/bin/printf 'user = "%s:%s"\n' "$username" "$password" |
      /usr/bin/curl --config - -sS --max-time 3 -o /dev/null -w '%{http_code}' \
        http://127.0.0.1:26001/
  )"
  unset password
  if [ "$authorized_status_code" != '200' ]; then
    echo "ERROR: authenticated monitor dashboard returned HTTP ${authorized_status_code}" >&2
    return 1
  fi
  trap - ERR
}

show_credentials_notice() {
  echo
  if [ "$password_generated" -eq 1 ]; then
    echo 'Fribench Monitor credentials were generated.'
    echo "Username: ${username}"
    echo "Password file: ${secret_file}"
    echo 'Retrieve the password now in this root terminal:'
    echo "  sudo /usr/bin/cat ${secret_file}"
    echo 'Do not save the password in Git, deployment notes, or chat.'
  else
    echo "Existing Fribench Monitor credentials were preserved in ${secret_file}."
  fi
}

main() {
  require_root
  validate_inputs
  prepare_secret
  install_ingress_assets
  activate_release
  /usr/local/sbin/fribench-direct-ports open
  show_credentials_notice
  echo
  echo 'Fribench Monitor authenticated public ingress is enabled on TCP 21002.'
}

main "$@"
