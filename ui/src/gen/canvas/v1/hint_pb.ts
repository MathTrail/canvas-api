// AUTO-GENERATED — do not edit manually.
// Run `just generate` in the contracts devcontainer to regenerate.
// buf.gen.yaml → buf.build/bufbuild/es → gen/ts/canvas/v1/
//
// This placeholder satisfies TypeScript until buf generate has been run.
// Replace this file with the generated output from contracts/gen/ts/canvas/v1/hint_pb.ts

export class HighlightRect {
  x: number = 0
  y: number = 0
  width: number = 0
  height: number = 0
  constructor(init?: Partial<HighlightRect>) { Object.assign(this, init) }
}

export class HintEvent {
  id: string = ''
  sessionId: string = ''
  hintText: string = ''
  rect: HighlightRect | undefined = undefined
  hintType: string = ''
  occurredAt: string = ''
  constructor(init?: Partial<HintEvent>) { Object.assign(this, init) }
  toBinary(): Uint8Array { throw new Error('replace with buf-generated file') }
  static fromBinary(_: Uint8Array): HintEvent { throw new Error('replace with buf-generated file') }
}
