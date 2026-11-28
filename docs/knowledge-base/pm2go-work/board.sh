#!/bin/sh
# Print the pm2go issue board from the per-issue files.
cd "$(dirname "$0")" || exit 1
printf '%-5s %-11s %-34s %-8s %s\n' ISSUE STATUS BRANCH PR TITLE
for f in [0-9][0-9][0-9]-*.md; do
  case "$f" in 000-*) continue;; esac
  n=$(head -1 "$f" | sed 's/^# #\([0-9]*\):.*/\1/')
  t=$(head -1 "$f" | sed 's/^# #[0-9]*: //')
  s=$(sed -n 's/^Status: *//p' "$f" | head -1)
  b=$(sed -n 's/^Branch: *//p' "$f" | head -1)
  p=$(sed -n 's/^PR: *//p' "$f" | head -1)
  printf '%-5s %-11s %-34s %-8s %s\n' "#$n" "$s" "$b" "$p" "$t"
done
