#!/usr/bin/env bash
# Regenerates plane-v1.4.2-schema.sql: runs the Django migrations of the
# Plane v1.4.2 backend image against an empty Postgres 15.7 and dumps the
# resulting schema. Needs only Docker (Compose 2.22 or later); see README.md.
set -euo pipefail

here="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
out="$here/plane-v1.4.2-schema.sql"
tmp="$out.tmp"
# Passed with -p on every call: it wins over COMPOSE_PROJECT_NAME, so the
# cleanup below can never touch another compose project.
project=nerve-plane-schema

die() {
  echo "extract.sh: $*" >&2
  exit 1
}

compose() {
  docker compose --project-name "$project" --file "$here/compose.yaml" "$@"
}

cleanup() {
  rm -f "$tmp"
  compose down --volumes --remove-orphans >/dev/null 2>&1 ||
    echo "extract.sh: cleanup failed; run: docker compose -p $project -f $here/compose.yaml down --volumes" >&2
}

command -v docker >/dev/null 2>&1 || die "docker not found; install Docker with Compose 2.22 or later"
docker info >/dev/null 2>&1 || die "cannot reach the Docker daemon; start Docker and retry"
docker compose version >/dev/null 2>&1 || die "docker compose not found; install Compose 2.22 or later"

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Leftovers of an interrupted (e.g. killed) run would otherwise be reused.
rm -f "$tmp"
compose down --volumes --remove-orphans >/dev/null 2>&1 || true

echo "==> pulling images (only those not present locally)"
compose pull --quiet --policy missing || die "pulling the images failed (needs Compose 2.22 or later)"

echo "==> starting Postgres"
compose up --detach --wait db || die "Postgres did not become healthy"

echo "==> running the Plane migrations"
# -T: without a TTY, Ctrl-C reaches this script (exit 130) instead of the container.
compose run --rm --no-deps -T migrator || die "the Plane migrations failed"

echo "==> dumping the schema"
compose exec -T db pg_dump --schema-only --no-owner --no-privileges --username plane --dbname plane >"$tmp" ||
  die "pg_dump failed"
mv "$tmp" "$out"

echo "==> wrote $out ($(wc -l <"$out" | tr -d ' ') lines)"
