#!/bin/sh
set -eu

release_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)"
release_prefix="/srv/fribench/apps/internal/fribench-monitor/releases/"
sudoers_source="${release_root}/deploy/fribench-monitor.sudoers"
sudoers_target="/etc/sudoers.d/fribench-monitor"

case "${release_root}/" in
  "${release_prefix}"*) ;;
  *)
    echo "Refusing to install from unexpected release path: ${release_root}" >&2
    exit 1
    ;;
esac

if [ ! -f "${sudoers_source}" ]; then
  echo "Missing sudoers source: ${sudoers_source}" >&2
  exit 1
fi

/usr/sbin/visudo -cf "${sudoers_source}"
sudo /usr/bin/install -o root -g root -m 0440 "${sudoers_source}" "${sudoers_target}"
sudo /usr/sbin/visudo -cf "${sudoers_target}"

echo "Installed restricted Fribench Monitor deployment permissions."
