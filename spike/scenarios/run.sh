#!/bin/bash
# Runs an end-to-end scenario against a freshly built server with
# authentication off. Usage: spike/scenarios/run.sh <scenario.sh> [workdir]
# The scenario script gets $BOT (platformbot binary), $WORK (work dir) and
# runs bots; this script builds, starts and stops the server.
set -u
root=$(cd "$(dirname "$0")/../.." && pwd)
scenario=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
WORK=${2:-$(mktemp -d)}
mkdir -p "$WORK/bin"
(cd "$root" && go build -o "$WORK/bin/dragonfactory" . && go build -o "$WORK/bin/platformbot" ./spike/cmd/platformbot) || exit 1
export BOT=$WORK/bin/platformbot WORK
cd "$WORK" || exit 1
cp -r "$root/assets" . 2>/dev/null
if [ ! -f config.toml ]; then
  "$WORK/bin/dragonfactory" > /dev/null 2>&1 & p=$!; sleep 2; kill -INT $p 2>/dev/null; wait $p
  sed -i 's/AuthEnabled = true/AuthEnabled = false/' config.toml
  # The host bot may use the operator tools.
  sed -i 's/^\(\s*\)Operators = \[\]/\1Operators = ["Hana"]/' platform.toml
fi
"$WORK/bin/dragonfactory" > srv.log 2>&1 & pid=$!
for _ in $(seq 1 100); do grep -q "server started" srv.log && break; sleep 0.2; done
bash "$scenario"; status=$?
kill -INT $pid; wait $pid
if grep -qiE "panic|ERROR" srv.log; then echo "server log has errors:"; grep -iE "panic|ERROR" srv.log | head; status=1; fi
echo "scenario exit $status (work dir $WORK)"
exit $status
