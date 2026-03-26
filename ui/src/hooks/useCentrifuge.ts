import { useEffect, useRef } from 'react'
// centrifuge-js Protobuf variant — binary WebSocket transport
import { Centrifuge } from 'centrifuge/build/protobuf'
import { HintEvent } from '../gen/canvas/v1/hint_pb'
import { useCanvasStore } from '../store/useCanvasStore'

interface UseCentrifugeProps {
  wsUrl: string
  token: string
  channel: string
  channelToken: string
}

export function useCentrifuge({ wsUrl, token, channel, channelToken }: UseCentrifugeProps) {
  const addServerHint = useCanvasStore((s) => s.addServerHint)
  const centrifugeRef = useRef<InstanceType<typeof Centrifuge> | null>(null)

  useEffect(() => {
    if (!token || !channel || !channelToken) return

    const c = new Centrifuge(wsUrl, { token })
    centrifugeRef.current = c

    const sub = c.newSubscription(channel, { token: channelToken })

    sub.on('publication', (ctx) => {
      try {
        const hint = HintEvent.fromBinary(new Uint8Array(ctx.data as ArrayBuffer))
        addServerHint({
          id: hint.id,
          hintText: hint.hintText,
          hintType: hint.hintType,
          rect: hint.rect
            ? {
                x: hint.rect.x,
                y: hint.rect.y,
                width: hint.rect.width,
                height: hint.rect.height,
              }
            : { x: 0, y: 0, width: 0, height: 0 },
        })
      } catch (e) {
        console.error('failed to decode HintEvent', e)
      }
    })

    sub.subscribe()
    c.connect()

    return () => {
      sub.unsubscribe()
      c.disconnect()
    }
  }, [wsUrl, token, channel, channelToken, addServerHint])
}
