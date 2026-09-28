# canvas-api

MathTrail Canvas is a math tutor for children in grades 1–6. A child solves a problem by hand, with a stylus or a finger, on a virtual sheet of paper. The service checks every step of the solution. When it finds a mistake, it does not give the answer: it highlights the spot and asks a short guiding question.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="draft/ui/screens/hint-dark.png">
  <img src="draft/ui/screens/hint-light.png" alt="MathTrail Canvas on an iPad in landscape. A child has written two steps for a triangle-angle task on grid paper. Line 1 is correct and has no marks. Line 2, 180 + 115 = 295, has a soft frame and a pointer in the margin. A speech bubble from the AI tutor, with its avatar, asks: can one angle of a triangle be bigger than 180?">
</picture>

What the child sees on the canvas:

- **Task layer.** The condition sits at the top of the page and never changes.
- **The child's ink.** Each step goes on its own ruled line. Line 1 is right, so the tutor stays silent.
- **Hint layer.** Line 2 uses the wrong operation. The AI tutor frames that line, points to it in the margin and asks one guiding question in a speech bubble. It never shows the answer and never covers the child's writing. If the child makes the same mistake again, the hints get more specific (up to three), and then the tutor suggests asking a grown-up.

This is a design mockup. The geometry task is only an example: the MVP ships with one equation and one arrow-matching task.

### Canvas and hint settings

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="draft/ui/screens/settings-dark.png">
  <img src="draft/ui/screens/settings-light.png" alt="The grown-up settings panel next to the canvas. Hint layer: highlight the step, pointer in the margin, question next to the line, overlay strength, marker colour. Checking: check after a pause. Paper: ruled, grid (selected) or dots, and line numbers. The page on the left shrinks to stay fully visible next to the panel.">
</picture>

Child mode has no settings of its own. A grown-up opens this panel with their PIN, and every change shows up on the page right away.

### Under the hood

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="draft/ui/screens/events-dark.png">
  <img src="draft/ui/screens/events-light.png" alt="The same screen with a developer inspector showing the CanvasEvent behind the hint: verdict line:2 ERROR WRONG_INVERSE_OPERATION, then a hint with teacherText, a highlight action and a point action with a rect in page units, source llm, level 1, kind hint.">
</picture>

What the client receives on the realtime channel `canvas:{canvas_id}` is a `hint` event. Its `highlight` and `point` actions carry a `rect` in logical page units: the page is 1000 units wide, so the frame lands on the same strokes on any screen. The event format is in [docs/spec.md](docs/spec.md) (in Russian).

To try the interactive mockup, open [draft/ui/index.html](draft/ui/index.html) in a browser. You can write with the pen, erase whole strokes and open **Grown-ups** with any 4 digits. Press `d` for the event inspector.
