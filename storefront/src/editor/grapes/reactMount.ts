import type { ReactElement } from 'react'
import { createRoot, type Root } from 'react-dom/client'

interface Mount {
  root: Root
  container: HTMLElement
}

/**
 * React roots for canvas components, one per component model. Each root
 * lives in its own container inside the GrapesJS element: GrapesJS empties
 * that element when it re-renders a view, which detaches the container
 * intact, so the old root can still be unmounted cleanly (unmounting a
 * root whose nodes were ripped out throws "removeChild"). Prop changes
 * re-render the existing root in place.
 */
export class ReactMounts<K extends object> {
  private readonly mounts = new WeakMap<K, Mount>()

  render(key: K, host: HTMLElement, element: ReactElement) {
    let mount = this.mounts.get(key)
    if (!mount || mount.container.parentNode !== host) {
      mount?.root.unmount()
      const container = host.ownerDocument.createElement('div')
      container.style.display = 'contents'
      host.appendChild(container)
      mount = { root: createRoot(container), container }
      this.mounts.set(key, mount)
    }
    mount.root.render(element)
  }

  unmount(key: K) {
    const mount = this.mounts.get(key)
    if (!mount) return
    mount.root.unmount()
    mount.container.remove()
    this.mounts.delete(key)
  }
}
