// AUTO-GENERATED — do not edit manually.
// Run `just generate` in the contracts devcontainer to regenerate.
// buf.gen.yaml → buf.build/bufbuild/es → gen/ts/canvas/v1/
//
// This placeholder satisfies TypeScript until buf generate has been run.
// Replace this file with the generated output from contracts/gen/ts/canvas/v1/stroke_pb.ts

export class Point {
  x: number = 0
  y: number = 0
  pressure: number = 0
  constructor(init?: Partial<Point>) { Object.assign(this, init) }
  toBinary(): Uint8Array { throw new Error('replace with buf-generated file') }
  static fromBinary(_: Uint8Array): Point { throw new Error('replace with buf-generated file') }
}

export class Stroke {
  id: string = ''
  points: Point[] = []
  tool: string = ''
  constructor(init?: Partial<Stroke>) { Object.assign(this, init) }
}

export class CanvasStrokeEvent {
  eventId: string = ''
  sessionId: string = ''
  taskId: string = ''
  userId: string = ''
  strokes: Stroke[] = []
  occurredAt: string = ''
  constructor(init?: Partial<CanvasStrokeEvent>) { Object.assign(this, init) }
  toBinary(): Uint8Array { throw new Error('replace with buf-generated file') }
  static fromBinary(_: Uint8Array): CanvasStrokeEvent { throw new Error('replace with buf-generated file') }
}
