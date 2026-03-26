import { create } from 'zustand'

export type Tool = 'pen' | 'eraser'

export interface RawStroke {
  id: string
  tool: Tool
  points: Array<{ x: number; y: number; pressure: number }>
  color: string
}

export interface ServerHint {
  id: string
  hintText: string
  hintType: string
  rect: { x: number; y: number; width: number; height: number }
}

interface CanvasState {
  currentTool: Tool
  strokes: RawStroke[]
  activeStroke: RawStroke | null
  serverHints: ServerHint[]

  setTool: (tool: Tool) => void
  beginStroke: (stroke: RawStroke) => void
  updateActiveStroke: (point: { x: number; y: number; pressure: number }) => void
  commitStroke: () => RawStroke | null
  clearStrokes: () => void
  addServerHint: (hint: ServerHint) => void
}

export const useCanvasStore = create<CanvasState>((set, get) => ({
  currentTool: 'pen',
  strokes: [],
  activeStroke: null,
  serverHints: [],

  setTool: (tool) => set({ currentTool: tool }),

  beginStroke: (stroke) => set({ activeStroke: stroke }),

  updateActiveStroke: (point) => {
    const { activeStroke } = get()
    if (!activeStroke) return
    set({
      activeStroke: {
        ...activeStroke,
        points: [...activeStroke.points, point],
      },
    })
  },

  commitStroke: () => {
    const { activeStroke } = get()
    if (!activeStroke || activeStroke.points.length < 2) {
      set({ activeStroke: null })
      return null
    }
    set((s) => ({
      strokes: [...s.strokes, activeStroke],
      activeStroke: null,
    }))
    return activeStroke
  },

  clearStrokes: () => set({ strokes: [], activeStroke: null }),

  addServerHint: (hint) =>
    set((s) => ({
      serverHints: [...s.serverHints.filter((h) => h.id !== hint.id), hint],
    })),
}))
