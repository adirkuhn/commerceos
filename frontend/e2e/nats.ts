import { connect } from 'node:net'

// Publishes one message to the local stack's NATS with the plain client
// protocol (HPUB), no client library: enough to put a dead letter on a DLQ
// stream, which stores whatever is published on its subjects. PING/PONG
// confirms the server processed the publish before returning.
export const NATS_URL = process.env.E2E_NATS_URL ?? 'nats://localhost:4222'

export async function publish(subject: string, data: string, headers: Record<string, string> = {}): Promise<void> {
  const { hostname, port } = new URL(NATS_URL)
  const hdr = `NATS/1.0\r\n${Object.entries(headers).map(([k, v]) => `${k}: ${v}\r\n`).join('')}\r\n`
  const hdrLen = Buffer.byteLength(hdr)
  const total = hdrLen + Buffer.byteLength(data)
  await new Promise<void>((resolve, reject) => {
    const socket = connect(Number(port) || 4222, hostname)
    let buffer = ''
    const timer = setTimeout(() => { socket.destroy(); reject(new Error('nats: no PONG within 5s')) }, 5000)
    socket.on('error', err => { clearTimeout(timer); reject(err) })
    socket.on('data', chunk => {
      buffer += chunk.toString()
      if (buffer.startsWith('INFO') && !buffer.includes('\r\n')) return
      if (buffer.includes('-ERR')) {
        clearTimeout(timer)
        socket.destroy()
        reject(new Error(`nats: ${buffer.trim()}`))
        return
      }
      if (buffer.includes('PONG')) {
        clearTimeout(timer)
        socket.end()
        resolve()
      }
    })
    socket.once('data', () => {
      socket.write(`CONNECT {"verbose":false,"pedantic":false,"headers":true,"no_responders":true}\r\n`)
      socket.write(`HPUB ${subject} ${hdrLen} ${total}\r\n${hdr}${data}\r\nPING\r\n`)
    })
  })
}
