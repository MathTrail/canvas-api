import React, { useCallback, useRef } from 'react'
import { Stage, Layer, Text, Shape, Line, Rect } from 'react-konva'
import type Konva from 'konva'
import { getStroke } from 'perfect-freehand'
import { useCanvasStore } from '../store/useCanvasStore'
import { sendStrokes } from '../transport/strokeApi'
import type { RawStroke } from '../store/useCanvasStore'

interface OlympiadCanvasProps {
  taskText: string
  sessionId: string
  taskId: string
  userId: string
  width?: number
  height?: number
}

const PEN_OPTIONS = { size: 5, thinning: 0.5, smoothing: 0.5, streamline: 0.5 }
const ERASER_OPTIONS = { size: 20, thinning: 0, smoothing: 0, streamline: 0 }

function strokeToPath(points: Array<{ x: number; y: number; pressure: number }>, isEraser: boolean) {
  const opts = isEraser ? ERASER_OPTIONS : PEN_OPTIONS
  return getStroke(
    points.map((p) => [p.x, p.y, p.pressure]),
    opts,
  )
}

export function OlympiadCanvas({
  taskText,
  sessionId,
  taskId,
  userId,
  width = 800,
  height = 500,
}: OlympiadCanvasProps) {
  const { currentTool, strokes, activeStroke, beginStroke, updateActiveStroke, commitStroke } =
    useCanvasStore()
  const serverHints = useCanvasStore((s) => s.serverHints)
  const isDrawing = useRef(false)

  const getPos = (e: Konva.KonvaEventObject<PointerEvent>) => {
    const stage = e.target.getStage()!
    const pos = stage.getPointerPosition()!
    const pressure = (e.evt as PointerEvent).pressure ?? 0.5
    return { x: pos.x, y: pos.y, pressure }
  }

  const handlePointerDown = useCallback(
    (e: Konva.KonvaEventObject<PointerEvent>) => {
      isDrawing.current = true
      const { x, y, pressure } = getPos(e)
      const stroke: RawStroke = {
        id: crypto.randomUUID(),
        tool: currentTool,
        points: [{ x, y, pressure }],
        color: currentTool === 'pen' ? '#1e1e1e' : 'transparent',
      }
      beginStroke(stroke)
    },
    [currentTool, beginStroke],
  )

  const handlePointerMove = useCallback(
    (e: Konva.KonvaEventObject<PointerEvent>) => {
      if (!isDrawing.current) return
      const point = getPos(e)
      updateActiveStroke(point)
    },
    [updateActiveStroke],
  )

  const handlePointerUp = useCallback(async () => {
    if (!isDrawing.current) return
    isDrawing.current = false
    const committed = commitStroke()
    if (committed && committed.tool === 'pen') {
      try {
        await sendStrokes(sessionId, taskId, userId, [committed])
      } catch (e) {
        console.error('sendStrokes error', e)
      }
    }
  }, [commitStroke, sessionId, taskId, userId])

  return (
    <Stage
      width={width}
      height={height}
      className="border border-slate-200 rounded-lg bg-white cursor-crosshair"
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
    >
      {/* Layer 1: Task — read-only, never erased */}
      <Layer listening={false}>
        <Text
          text={taskText}
          x={32}
          y={32}
          fontSize={36}
          fontFamily="serif"
          fill="#374151"
        />
      </Layer>

      {/* Layer 2: Drawing — pen strokes + eraser */}
      <Layer>
        {strokes.map((stroke) => {
          const outline = strokeToPath(stroke.points, stroke.tool === 'eraser')
          if (stroke.tool === 'eraser') {
            // Eraser: plain Line with destination-out — only erases within this canvas element
            return (
              <Line
                key={stroke.id}
                points={stroke.points.flatMap((p) => [p.x, p.y])}
                stroke="rgba(0,0,0,1)"
                strokeWidth={ERASER_OPTIONS.size}
                lineCap="round"
                lineJoin="round"
                globalCompositeOperation="destination-out"
              />
            )
          }
          // Pen: Shape with sceneFunc — fill the perfect-freehand outline
          return (
            <Shape
              key={stroke.id}
              sceneFunc={(ctx, shape) => {
                if (outline.length < 2) return
                ctx.beginPath()
                outline.forEach(([x, y], i) => {
                  if (i === 0) ctx.moveTo(x, y)
                  else ctx.lineTo(x, y)
                })
                ctx.closePath()
                ctx.fillStrokeShape(shape)
              }}
              fill={stroke.color}
            />
          )
        })}

        {/* Active (in-progress) stroke */}
        {activeStroke && activeStroke.points.length > 1 && (() => {
          const outline = strokeToPath(activeStroke.points, activeStroke.tool === 'eraser')
          if (activeStroke.tool === 'eraser') {
            return (
              <Line
                key="active"
                points={activeStroke.points.flatMap((p) => [p.x, p.y])}
                stroke="rgba(0,0,0,1)"
                strokeWidth={ERASER_OPTIONS.size}
                lineCap="round"
                lineJoin="round"
                globalCompositeOperation="destination-out"
              />
            )
          }
          return (
            <Shape
              key="active"
              sceneFunc={(ctx, shape) => {
                if (outline.length < 2) return
                ctx.beginPath()
                outline.forEach(([x, y], i) => {
                  if (i === 0) ctx.moveTo(x, y)
                  else ctx.lineTo(x, y)
                })
                ctx.closePath()
                ctx.fillStrokeShape(shape)
              }}
              fill={activeStroke.color}
            />
          )
        })()}
      </Layer>

      {/* Layer 3: Hints — server-pushed overlays */}
      <Layer listening={false}>
        {serverHints.map((hint) => (
          <React.Fragment key={hint.id}>
            <Rect
              x={hint.rect.x}
              y={hint.rect.y}
              width={hint.rect.width}
              height={hint.rect.height}
              fill={hint.hintType === 'success' ? 'rgba(34,197,94,0.2)' : 'rgba(239,68,68,0.2)'}
              stroke={hint.hintType === 'success' ? '#22c55e' : '#ef4444'}
              strokeWidth={2}
              cornerRadius={4}
            />
            <Text
              text={hint.hintText}
              x={hint.rect.x}
              y={hint.rect.y + hint.rect.height + 4}
              fontSize={13}
              fill={hint.hintType === 'success' ? '#15803d' : '#b91c1c'}
              fontFamily="sans-serif"
            />
          </React.Fragment>
        ))}
      </Layer>
    </Stage>
  )
}

