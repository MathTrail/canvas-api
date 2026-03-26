import { Pencil, Eraser, Trash2 } from 'lucide-react'
import { useCanvasStore, type Tool } from '../store/useCanvasStore'

export function Toolbar() {
  const currentTool = useCanvasStore((s) => s.currentTool)
  const setTool = useCanvasStore((s) => s.setTool)
  const clearStrokes = useCanvasStore((s) => s.clearStrokes)

  const btnClass = (tool: Tool) =>
    `flex items-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium transition-colors ` +
    (currentTool === tool
      ? 'bg-slate-900 text-white'
      : 'bg-white text-slate-700 border border-slate-200 hover:bg-slate-50')

  return (
    <div className="flex items-center gap-2 p-2 bg-slate-50 border-b border-slate-200">
      <button className={btnClass('pen')} onClick={() => setTool('pen')}>
        <Pencil size={16} />
        Pen
      </button>
      <button className={btnClass('eraser')} onClick={() => setTool('eraser')}>
        <Eraser size={16} />
        Eraser
      </button>
      <div className="ml-auto">
        <button
          className="flex items-center gap-1.5 px-3 py-2 rounded-md text-sm font-medium text-red-600 border border-red-200 hover:bg-red-50 transition-colors"
          onClick={clearStrokes}
        >
          <Trash2 size={16} />
          Clear
        </button>
      </div>
    </div>
  )
}
