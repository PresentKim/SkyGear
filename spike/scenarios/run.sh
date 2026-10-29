#!/bin/bash
# Runs an end-to-end scenario against a freshly built server with
# authentication off. Usage: [PORT=n] spike/scenarios/run.sh <scenario.sh> [workdir]
# The scenario script gets $BOT (platformbot binary), $WORK (work dir) and
# runs bots; this script builds, starts and stops the server.
set -u
root=$(cd "$(dirname "$0")/../.." && pwd)
scenario=$(cd "$(dirname "$1")" && pwd)/$(basename "$1")
WORK=${2:-$(mktemp -d)}
mkdir -p "$WORK/bin"
# Native Windows bots get paths inside their arguments unconverted: give
# them a C:/... path.
command -v cygpath > /dev/null && WORK=$(cygpath -m "$WORK")
# The server is built at one fixed path (the repo's own build output), so a
# firewall rule allowing it keeps working for every work dir.
server=$root/skygear$(go env GOEXE)
(cd "$root" && go build -o "$server" . && go build -o "$WORK/bin/platformbot" ./spike/cmd/platformbot) || exit 1
# stop interrupts the server so it saves, and kills it if it is still
# running after 15s (Git Bash on Windows cannot interrupt a native process).
stop() {
  kill -INT "$1" 2>/dev/null
  for _ in $(seq 1 75); do kill -0 "$1" 2>/dev/null || break; sleep 0.2; done
  kill "$1" 2>/dev/null
  wait "$1"
}
export BOT=$WORK/bin/platformbot WORK
cd "$WORK" || exit 1
if [ ! -f config.toml ]; then
  "$server" > /dev/null 2>&1 & p=$!; sleep 2; stop $p
  sed -i 's/AuthEnabled = true/AuthEnabled = false/' config.toml
  # The host bot may use the operator tools.
  sed -i 's/^\(\s*\)Operators = \[\]/\1Operators = ["Hana"]/' platform.toml
fi
# PORT runs the server and the bots on another port (another server on 19132).
port=${PORT:-19132}
sed -i "s/^\(\s*Address = \"\):[0-9]*\"/\1:$port\"/" config.toml
export BOT_ADDR=127.0.0.1:$port
"$server" > srv.log 2>&1 & pid=$!
for _ in $(seq 1 100); do grep -q "server started" srv.log && break; sleep 0.2; done
bash "$scenario"; status=$?
stop $pid
if grep -qiE "panic|ERROR" srv.log; then echo "server log has errors:"; grep -iE "panic|ERROR" srv.log | head; status=1; fi
echo "scenario exit $status (work dir $WORK)"
exit $status
