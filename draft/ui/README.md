# Canvas mockup

A clickable mockup of the MathTrail Canvas screen in child mode, on an iPad in landscape. It is one self-contained file, [index.html](index.html). It uses system fonts and makes no network requests. Colours, type and spacing come from the MathTrail design system. The screenshots in [screens/](screens/) are embedded in the root [README](../../README.md).

## Open it

Open `index.html` in a browser. The iPad frame scales to fit the window.

- **Pen, blue pen:** write on the page.
- **Eraser:** removes a whole stroke, as in the spec. Erasing any part of line 2 clears the hint, as it would once the child fixes the line.
- **Undo / Redo:** work for both writing and erasing.
- **Scrolling:** the page has 14 lines. Scroll with the mouse wheel, a touchpad or two fingers, or tap the page map on the right to jump. The fit button goes back to the top.
- **Check:** or pause for 1.5 s after a stroke. You'll see *Checking…*, then nothing, because a correct step gets no comment.
- **The tutor's help:** the bar next to the tutor's icon in the top bar fills as the day's help is used. When it is full, nothing is checked: the hint layer stays empty, the tutor says once at the bottom of the page that the child can keep going alone, and Check repeats it. The note fades after a few seconds and never takes a stroke.
- **Grown-ups:** opens the PIN pad (any 4 digits), then the plan and the canvas and hint settings. **Upgrade** switches the plan to Plus.
- **`d`:** shows the event inspector. Its buttons switch the hint ladder (1, 2, 3, help) and the theme.
- **`q`:** fills the help bar: 90%, used up, back to 40%. **`p`:** switches the plan between Free and Plus.
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
| `quota` | the share of today's help already used, `0`–`100` (default `40`); `100` shows the help used up, with line 2 as if written after that and so never checked |
| `plan` | `free` (default), `plus` |

## Regenerate the screenshots

Run [screens.sh](screens.sh). It needs only Docker and the host's Noto Sans, the font the committed images use; without the font it stops.

- Firefox from the Playwright image takes all eight screenshots with its own `--screenshot`, with overlay scrollbars as on the iPad.
- Pillow saves them again as RGB with `optimize=True`, which makes them about 30% smaller.

Both images are pinned by digest. The stage is 1482 × 1140 and scales to the window, so the 2964 × 2280 window gives 2× images.

## What comes from the docs and what is proposed

**From the docs** ([product.md](../../docs/product.md), [spec.md](../../docs/spec.md)):

- **The page:** three layers (task, the child's ink, hints). It is 1000 logical units wide, with 14 ruled lines of 100 units starting at y = 200.
- **The marker:** a frame around the line's strokes plus 10 units, drawn from the `highlight` and `point` actions with target `line:N` and style `attention`.
- **The hint:** one question of at most two sentences, starting with what went right and never giving the answer. The ladder has three levels, then `offer_help` with *Start again* / *Ask a grown-up*.
- **Checking:** runs after a 1.5 s pause or on Check. A correct step gets silence.
- **Pen and eraser:** the eraser removes whole strokes. After the first Apple Pencil touch, the palm and fingers stop drawing.
- **Child mode:** exit needs the PIN. The child appears only as a nickname and an avatar. The UI is in English.
- **The day's help used up** ([R35](../../docs/decisions.md)): until it renews nothing is checked — no marks, questions or congratulations — and the child keeps writing. The tutor says so once, and again on Check. The inspector shows the verdict `UNCHECKED` with the reason `limit`.
- **Plans after the MVP** ([R36](../../docs/decisions.md), product 14): free; $10 a month for bigger quotas; $20 a month for even bigger quotas and an AI tutor with a voice. The MVP itself is a free closed beta (R05).

**Proposals, not yet in the docs:**

- A geometry task. The MVP has an equation and an arrow-matching task.
- The settings panel, apart from the pause, and everything in it: paper type, highlight and pointer toggles, where the question sits, overlay strength, marker colour, left-handed layout, pen thickness, larger text, reduced motion. The spec's child mode has no settings, so the panel sits behind the PIN.
- The blue pen, undo/redo, the page map, and the tutor bubble's position next to the line.
- The AI tutor's avatar and the "AI tutor" name on its questions. The avatar is a mortarboard and a spark on a disc in the hint's warm colours. It is not a mascot: the design system keeps the MathTrail mark for the app and a neutral figure for the child.
- The help meter: a bar with the tutor's icon that fills as the day's help is used. The child sees it in the top bar, with no numbers or prices; the grown-up sees it in the settings, with the share used. The docs have daily limits on recognitions, hints and drawings (spec 14.5), but no meter that shows them.
- The plan card in the settings, with Upgrade behind the PIN and the name Plus for the $10 plan. It shows the free plan and the $10 one; the $20 plan is not in the mockup. "Five times the daily help" is a placeholder: the quotas are still open (О-18 in product 15.2).
