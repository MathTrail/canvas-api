# The screen mockup

`MathTrail Minimal.html` is a clickable mockup of the task screen in child mode, made by the author before the LLM-first pivot of 2026-10-11 (R37). Open it in a browser: it runs on its own, with no server. It is a proposal, not a spec. Where it differs from the documents, the documents win.

## What it shows

- The screen: the task's condition at the top, a draft sheet, and an answer box with a **Check** button.
- A pen and an eraser on the sheet and in the answer box.
- **Check** sends the answer box as a PNG to a simulated checker. The result says what was read — "I read “6”" — and then either "Correct!" with the explanation of the solution, or "Not yet" with the author's text for that wrong answer.
- Tweaks to play with: the task (six of them), how the checker reads (right, a mistake, unreadable), and the failures to simulate (a slow network, a MathTrail error, a checker error).
- A log of the requests the client makes.

## Where the documents differ

- **The checker.** In the mockup, one simulated call reads the answer and judges it. In the product, Gemini only reads the answer, and a program compares it with the task's key (R38). After a wrong answer, a second call reads the draft together with the key, and the tutor frames the place of the mistake over the child's own strokes and asks one question (R39, R43). The mockup has no draft analysis and no frame.
- **The hint ladder.** The mockup shows the author's text for a wrong answer, every time. In the product, the questions climb a three-step ladder by the number of wrong answers on the sheet; after it, the tutor offers to ask a grown-up or start again, and the answer is never shown (R39, product 5.4).
- **The tasks.** The mockup fetches tasks from MathTrail through an MCP tool, `get_task`. The product has five static tasks in `tasks/` and no link to MathTrail. The task about turning cups over is not one of them (R40).
- **The eraser.** The mockup's eraser cuts strokes like pixels. The product's eraser removes whole strokes, because the frame of a mistake binds to whole strokes (R42, R43).
- **The draft.** The mockup keeps the sheet in the page's memory. The product keeps the strokes in IndexedDB on the device and sends the answer box as a PNG on every Check, and the draft too unless the sheet is empty (R42, spec 4.3).
- **Accepted answers.** The mockup accepts lists of words, such as `no` and `cannot`. In the product, an answer is a number or one of the task's choices, and its form is checked against a whitelist (spec 5.2).
- **Words on the screen.** The product's lines for every verdict, a failure and the daily limit are in product 5.5.
