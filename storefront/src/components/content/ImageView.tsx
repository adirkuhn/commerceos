import { storeHref } from '../../runtime/page/links'
import { resolveAssetRef, useRenderEnv } from '../../runtime/RenderContext'
import type { RendererProps } from '../types'
import type { ImageProps } from './image'

export function ImageView({ node, props, className }: RendererProps<ImageProps>) {
  const env = useRenderEnv()
  const asset = resolveAssetRef(props.image, env.assets)
  if (!asset) {
    return env.mode === 'editor'
      ? <div className={`cos-placeholder cos-image-placeholder ${className}`} data-node={node.id}>Choose an image</div>
      : null
  }
  const img = (
    <img src={asset.url} alt={props.alt || asset.alt} loading="lazy" decoding="async"
      width={asset.width || undefined} height={asset.height || undefined}
      className={props.fit === 'contain' ? 'cos-fit-contain' : 'cos-fit-cover'} />
  )
  const href = env.mode === 'live' ? storeHref(props.link, env.storePath) : undefined
  return (
    <figure className={`cos-image ${className}`} data-node={node.id}>
      {href ? <a href={href}>{img}</a> : img}
    </figure>
  )
}
