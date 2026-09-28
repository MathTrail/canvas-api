#!/bin/sh
# Regenerates the screenshots in screens/, which the root README embeds, in both themes.
# Only Docker runs on the host: the browser and the image tools come in images pinned by digest.
# The page uses system fonts, and the committed images use Noto Sans, so the host's copy is
# mounted into the browser's container (on Debian and Ubuntu it comes with fonts-noto-core).
set -eu
cd "$(dirname "$0")"

PLAYWRIGHT=mcr.microsoft.com/playwright:v1.63.0-noble@sha256:eff16c30e6f3f4af0a03fa4b706120d5e9b0891c344a27d64559aff5900a4a27
PYTHON=python:3.13.15-slim-trixie@sha256:7c61056e61ac89e852de05f3dc6fa51a6dd2181797bceed46aa725dd7cb2cd3b
NOTO=/usr/share/fonts/truetype/noto
USER_ID="$(id -u):$(id -g)"

# Without the font the images would come out in another typeface and nothing would say so
if [ ! -f "$NOTO/NotoSans-Regular.ttf" ]; then
  echo "screens.sh: Noto Sans is not in $NOTO; the committed screenshots use it" >&2
  exit 1
fi

# Firefox's own headless screenshot, from the Playwright image. The stage is 1482 × 1140 and
# scales to the window, so a 2964 × 2280 window gives 2× images. Overlay scrollbars, as on the
# iPad: a classic one would narrow the inspector and wrap its lines differently.
for theme in light dark; do
  for shot in "hint:scene=hint" "settings:scene=settings" "events:scene=hint&dev=1" "resting:scene=hint&quota=100"; do
    docker run --rm --init --user "$USER_ID" -e HOME=/tmp -v "$PWD:/work" \
      --mount "type=bind,src=$NOTO,dst=$NOTO,readonly" \
      --mount "type=bind,src=$PWD/fonts.conf,dst=/etc/fonts/local.conf,readonly" \
      "$PLAYWRIGHT" sh -c 'firefox=$(ls -d /ms-playwright/firefox-*/firefox)/firefox && profile=$(mktemp -d) &&
        printf "%s\n" "user_pref(\"widget.gtk.overlay-scrollbars.enabled\", true);" "user_pref(\"ui.useOverlayScrollbars\", 1);" >"$profile/user.js" &&
        "$firefox" --headless --no-remote --profile "$profile" --window-size=2964,2280 \
          --screenshot "/work/screens/$1-$2.png" "file:///work/index.html?$3&theme=$2" >/dev/null 2>&1 &&
        test -s "/work/screens/$1-$2.png"' \
      sh "${shot%%:*}" "$theme" "${shot#*:}"
  done
done

# Saved again as RGB with optimize=True, the images come out about 30% smaller.
docker run --rm -i --user "$USER_ID" -e HOME=/tmp -v "$PWD/screens:/screens" \
  "$PYTHON" sh -c 'pip install --quiet --no-cache-dir --disable-pip-version-check --target /tmp/pp pillow==12.3.0 </dev/null && PYTHONPATH=/tmp/pp python -' <<'EOF'
from pathlib import Path
from PIL import Image

for path in sorted(Path("/screens").glob("*.png")):
    Image.open(path).convert("RGB").save(path, optimize=True)
EOF
