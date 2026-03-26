import { useEffect, useState } from 'react'

export interface CanvasToken {
  token: string
  channel: string
  channelToken: string
}

export function useCanvasToken(sessionId: string): {
  tokenData: CanvasToken | null
  error: string | null
} {
  const [tokenData, setTokenData] = useState<CanvasToken | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!sessionId) return

    fetch(`/api/canvas/token?session_id=${encodeURIComponent(sessionId)}`, {
      credentials: 'include',
    })
      .then((r) => {
        if (!r.ok) throw new Error(`token fetch failed: ${r.status}`)
        return r.json()
      })
      .then((data) =>
        setTokenData({
          token: data.token,
          channel: data.channel,
          channelToken: data.channel_token,
        }),
      )
      .catch((e: unknown) => setError(String(e)))
  }, [sessionId])

  return { tokenData, error }
}
