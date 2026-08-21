#!/usr/bin/env fish

set -l root (dirname (status --current-filename))/..
set -l envfile $root/.env

if not test -f $envfile
    echo "missing $envfile — copy .env.example and fill it in" >&2
    exit 1
end

for line in (grep -vE '^\s*(#|$)' $envfile)
    set -l parts (string split -m1 '=' $line)
    if test (count $parts) -ne 2
        echo "ignoring malformed line: $line" >&2
        continue
    end
    set -gx $parts[1] $parts[2]
end

cd $root
exec go run ./cmd/debian-watch $argv
