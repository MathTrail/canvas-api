# Canvas mockup

A clickable mockup of the MathTrail Canvas screen in child mode, on an iPad in landscape. It is one self-contained file, [index.html](index.html). It uses system fonts and makes no network requests. Colours, type and spacing come from the MathTrail design system. The screenshots in [screens/](screens/) are embedded in the root [README](../../README.md).

## Open it

Open `index.html` in a browser. The iPad frame scales to fit the window.

- **Pen, blue pen:** write on the page.
- **Eraser:** removes a whole stroke, as in the spec. Erasing any part of line 2 clears the hint, as it would once the child fixes the line.
- **Undo / Redo:** work for both writing and erasing.
- **Scrolling:** the page has 14 lines. Scroll with the mouse wheel, a touchpad or two fingers, or tap the page map on the right to jump. The fit button goes back to the top.
- **Check:** or pause for 1.5 s after a stroke. You'll see *Checking…*, then nothing, because a correct step gets no comment.
- **Grown-ups:** opens the PIN pad (any 4 digits), then the canvas and hint settings.
- **`d`:** shows the event inspector. Its buttons switch the hint ladder (1, 2, 3, help) and the theme.
- **`t`:** switches the theme. **`Esc`:** closes the panels.

URL parameters, used for the screenshots:

| Parameter | Values |
|---|---|
| `scene` | `hint` (default), `settings`, `pin`, `cleared` |
| `theme` | `light`, `dark`; by default follows the system |
| `level` | `1`, `2`, `3`, `help` |
| `dev` | `1` shows the event inspector |
| `paper` | `grid` (default), `ruled`, `dots` |
| `scroll` | scroll the page to this y, in logical units (`700` shows lines 6–14) |
| `marker` | `warm`, `blue`, `contrast` |
| `bubble` | `0` moves the question to a bar at the bottom of the page |

## Regenerate the screenshots

The stage is 1482 × 1140 and scales to the window. A 2964 × 2280 window gives 2× images.

```sh
cd draft/ui
for t in light dark; do
  for s in "hint:scene=hint" "settings:scene=settings" "events:scene=hint&dev=1"; do
    firefox --headless --no-remote --profile "$(mktemp -d)" --window-size=2964,2280 \
      --screenshot "$PWD/screens/${s%%:*}-$t.png" "file://$PWD/index.html?${s#*:}&theme=$t"
  done
done
```

The committed PNGs were also saved as RGB with `optimize=True` in Pillow, which makes them about 30% smaller.

## What comes from the docs and what is proposed

**From the docs** ([product.md](../../docs/product.md), [spec.md](../../docs/spec.md)):

- **The page:** three layers (task, the child's ink, hints). It is 1000 logical units wide, with 14 ruled lines of 100 units starting at y = 200.
- **The marker:** a frame around the line's strokes plus 10 units, drawn from the `highlight` and `point` actions with target `line:N` and style `attention`.
- **The hint:** one question of at most two sentences, starting with what went right and never giving the answer. The ladder has three levels, then `offer_help` with *Start again* / *Ask a grown-up*.
- **Checking:** runs after a 1.5 s pause or on Check. A correct step gets silence.
- **Pen and eraser:** the eraser removes whole strokes. After the first Apple Pencil touch, the palm and fingers stop drawing.
- **Child mode:** exit needs the PIN. The child appears only as a nickname and an avatar. The UI is in English.

**Proposals, not yet in the docs:**

- A geometry task. The MVP has an equation and an arrow-matching task.
- The settings panel, apart from the pause, and everything in it: paper type, highlight and pointer toggles, where the question sits, overlay strength, marker colour, left-handed layout, pen thickness, larger text, reduced motion. The spec's child mode has no settings, so the panel sits behind the PIN.
- The blue pen, undo/redo, the page map, and the tutor bubble's position next to the line.
- The AI tutor's avatar and the "AI tutor" name on its questions. The avatar is a mortarboard and a spark on a disc in the hint's warm colours. It is not a mascot: the design system keeps the MathTrail mark for the app and a neutral figure for the child.
