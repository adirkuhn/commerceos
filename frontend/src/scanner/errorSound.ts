// Short square-wave error beep, generated in-browser via the Web Audio API
// -- no audio asset or npm package needed (see CLAUDE.md's third-party
// approval rule). Shared by both the web Picking page and the scanner
// app's PickingScreen so a mis-scan/over-scan alert sounds the same on
// both. Best-effort: a blocked or unsupported AudioContext (e.g. jsdom in
// tests, or a browser that hasn't granted audio permission yet) must never
// crash the scan flow.
export function playErrorSound(): void {
  try {
    const AudioContextCtor = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!AudioContextCtor) return
    const ctx = new AudioContextCtor()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.type = 'square'
    osc.frequency.value = 220
    gain.gain.setValueAtTime(0.2, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 0.3)
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.start()
    osc.stop(ctx.currentTime + 0.3)
  } catch {
    // best-effort, see doc comment above
  }
}
