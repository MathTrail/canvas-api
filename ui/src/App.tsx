import { OlympiadCanvas } from './components/OlympiadCanvas'
import { Toolbar } from './components/Toolbar'
import { useCanvasToken } from './hooks/useCanvasToken'
import { useCentrifuge } from './hooks/useCentrifuge'

// Dev entry point: hardcoded task "1 + 1 ="
// In production this component is consumed by ui-web shell via Module Federation.
const DEV_SESSION_ID = 'dev-session-001'
const DEV_TASK_ID = 'dev-task-001'
const DEV_USER_ID = 'dev-user-001'
const DEV_TASK_TEXT = '1 + 1 ='
const WS_URL = `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/connection/websocket`

export default function App() {
  const { tokenData, error } = useCanvasToken(DEV_SESSION_ID)

  useCentrifuge({
    wsUrl: WS_URL,
    token: tokenData?.token ?? '',
    channel: tokenData?.channel ?? '',
    channelToken: tokenData?.channelToken ?? '',
  })

  return (
    <div className="min-h-screen bg-slate-100 flex flex-col items-center justify-center p-6">
      <div className="bg-white rounded-xl shadow-sm overflow-hidden w-[860px]">
        <Toolbar />
        <div className="p-4">
          <OlympiadCanvas
            taskText={DEV_TASK_TEXT}
            sessionId={DEV_SESSION_ID}
            taskId={DEV_TASK_ID}
            userId={DEV_USER_ID}
            width={828}
            height={500}
          />
        </div>
      </div>
      {error && (
        <p className="mt-3 text-sm text-red-500">Canvas token error: {error}</p>
      )}
    </div>
  )
}
