import { CanvasStrokeEvent, Stroke, Point } from '../gen/canvas/v1/stroke_pb'
import type { RawStroke } from '../store/useCanvasStore'

export async function sendStrokes(
  sessionId: string,
  taskId: string,
  userId: string,
  rawStrokes: RawStroke[],
): Promise<void> {
  const strokes = rawStrokes.map((rs) => {
    const points = rs.points.map((p) =>
      new Point({ x: p.x, y: p.y, pressure: p.pressure }),
    )
    return new Stroke({ id: rs.id, points, tool: rs.tool })
  })

  const event = new CanvasStrokeEvent({
    eventId: crypto.randomUUID(),
    sessionId,
    taskId,
    userId,
    strokes,
    occurredAt: new Date().toISOString(),
  })

  const body = event.toBinary()

  const res = await fetch('/api/canvas/strokes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/octet-stream' },
    body,
    credentials: 'include', // forward Ory session cookie
  })

  if (!res.ok) {
    throw new Error(`sendStrokes failed: ${res.status}`)
  }
}
