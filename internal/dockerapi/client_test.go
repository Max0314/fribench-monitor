package dockerapi

import "testing"

func TestSafeCommandNameOmitsArguments(t *testing.T) {
	command := `docker-entrypoint.sh /bin/sh -ec 'exec redis-server --requirepass "$(cat /run/secrets/redis_password)"'`
	if got := safeCommandName(command); got != "docker-entrypoint.sh" {
		t.Fatalf("safe command = %q", got)
	}
}

func TestSafeCommandNameUsesExecutableBaseName(t *testing.T) {
	if got := safeCommandName(`/app/continuity-server --config /run/secrets/config`); got != "continuity-server" {
		t.Fatalf("safe command = %q", got)
	}
}
