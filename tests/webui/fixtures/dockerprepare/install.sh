
stage=$1
user=$2
for parent in /data /data/chainbench /data/chainbench/keys /root; do
  [ ! -L "$parent" ] || exit 20
  if [ -e "$parent" ]; then [ -d "$parent" ] || exit 20; fi
done
chown -R "$user:" "$stage/kr-src" "$stage/nodelog-test.log"
chmod 644 "$stage/nodelog-test.log"
find "$stage/kr-src" -type d -exec chmod 755 {} +
find "$stage/kr-src" -type f -exec chmod 644 {} +
chown -R root:root "$stage/cb-kr-test" "$stage/cb-sudo-test.txt"
find "$stage/cb-kr-test" -type d -exec chmod 700 {} +
find "$stage/cb-kr-test" -type f -exec chmod 600 {} +
chmod 600 "$stage/cb-sudo-test.txt"
check() {
  src=$1 dst=$2
  [ ! -L "$dst" ] || exit 21
  if [ -e "$dst" ]; then
    diff -qr --no-dereference "$src" "$dst" >/dev/null || exit 22
    if [ -d "$src" ]; then
      [ -d "$dst" ] || exit 22
      (cd "$src"; find . -printf '%P %m %U %G %y\n' | LC_ALL=C sort) > "$stage/expected"
      (cd "$dst"; find . -printf '%P %m %U %G %y\n' | LC_ALL=C sort) > "$stage/actual"
      cmp "$stage/expected" "$stage/actual" || exit 23
    else
      [ "$(stat -c '%a %u %g %F' "$src")" = "$(stat -c '%a %u %g %F' "$dst")" ] || exit 23
    fi
  fi
}
check "$stage/kr-src" /data/chainbench/keys/kr-src
check "$stage/nodelog-test.log" /data/chainbench/nodelog-test.log
check "$stage/cb-sudo-test.txt" /root/cb-sudo-test.txt
check "$stage/cb-kr-test" /root/cb-kr-test
for parent in /data/chainbench /data/chainbench/keys; do
  if [ ! -e "$parent" ]; then mkdir "$parent"; chown "$user:" "$parent"; fi
done
install_missing() { if [ ! -e "$2" ]; then mv -T -n "$1" "$2"; fi; }
install_missing "$stage/kr-src" /data/chainbench/keys/kr-src
install_missing "$stage/nodelog-test.log" /data/chainbench/nodelog-test.log
install_missing "$stage/cb-sudo-test.txt" /root/cb-sudo-test.txt
install_missing "$stage/cb-kr-test" /root/cb-kr-test
