import { useEffect } from 'react'

interface SeoProps {
  title: string
  description?: string
  image?: string
  jsonLd?: Record<string, unknown> | Record<string, unknown>[]
}

const SITE_NAME = 'Aurora'

function setMeta(name: string, content: string, attr: 'name' | 'property' = 'name') {
  let tag = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${name}"]`)
  if (!tag) {
    tag = document.createElement('meta')
    tag.setAttribute(attr, name)
    document.head.appendChild(tag)
  }
  tag.setAttribute('content', content)
}

function setCanonical(href: string) {
  let link = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]')
  if (!link) {
    link = document.createElement('link')
    link.setAttribute('rel', 'canonical')
    document.head.appendChild(link)
  }
  link.setAttribute('href', href)
}

// Hand-rolled document-head manager -- no react-helmet (unapproved
// dependency, and this app has no SSR yet so a single useEffect covers
// every route). Sets title/meta description/OG/Twitter/canonical, plus
// optional JSON-LD structured data (see features/products'
// ProductDetailPage for a real Product + BreadcrumbList example).
export function Seo({ title, description, image, jsonLd }: SeoProps) {
  useEffect(() => {
    const fullTitle = `${title} | ${SITE_NAME}`
    document.title = fullTitle
    if (description) {
      setMeta('description', description)
      setMeta('og:description', description, 'property')
      setMeta('twitter:description', description)
    }
    setMeta('og:title', fullTitle, 'property')
    setMeta('og:type', 'website', 'property')
    setMeta('twitter:card', image ? 'summary_large_image' : 'summary')
    setMeta('twitter:title', fullTitle)
    if (image) {
      setMeta('og:image', image, 'property')
      setMeta('twitter:image', image)
    }
    setCanonical(window.location.href)

    let script: HTMLScriptElement | null = null
    if (jsonLd) {
      script = document.createElement('script')
      script.type = 'application/ld+json'
      script.textContent = JSON.stringify(jsonLd)
      document.head.appendChild(script)
    }
    return () => {
      if (script) document.head.removeChild(script)
    }
  }, [title, description, image, jsonLd])

  return null
}
