// GrapesJS ships its UI translations as plain modules without types.
declare module 'grapesjs/locale/*.mjs' {
  const messages: Record<string, unknown>
  export default messages
}
