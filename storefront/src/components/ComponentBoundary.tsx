import { Component, type ReactNode } from 'react'

interface Props {
  nodeId: string
  /** Shown instead of the component; the live store shows nothing. */
  fallback?: ReactNode
  children: ReactNode
}

/** One failing component (e.g. its data didn't load) must not take the
 *  page -- or the editor canvas -- down with it. */
export class ComponentBoundary extends Component<Props, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  componentDidCatch(error: unknown) {
    console.error(`storefront: component ${this.props.nodeId} failed`, error)
  }

  render() {
    return this.state.failed ? (this.props.fallback ?? null) : this.props.children
  }
}
