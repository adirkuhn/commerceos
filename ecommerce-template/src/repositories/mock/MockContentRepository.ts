import type { ContentRepository } from '../types'
import { simulateLatency } from './latency'
import { CMS_PAGES, BLOG_POSTS, FAQ_ITEMS } from '@/config/demoData'

export class MockContentRepository implements ContentRepository {
  async getPage(slug: string) {
    await simulateLatency()
    return CMS_PAGES.find(p => p.slug === slug) ?? null
  }

  async listBlogPosts() {
    await simulateLatency()
    return [...BLOG_POSTS].sort((a, b) => new Date(b.publishedAt).getTime() - new Date(a.publishedAt).getTime())
  }

  async getBlogPost(slug: string) {
    await simulateLatency()
    return BLOG_POSTS.find(p => p.slug === slug) ?? null
  }

  async faq() {
    await simulateLatency()
    return FAQ_ITEMS
  }
}
