import { useQuery } from '@tanstack/react-query'
import { getCmsPage, listBlogPosts, getBlogPost, listFaqItems } from '@/api/content'

export function useCmsPage(slug: string | undefined) {
  return useQuery({ queryKey: ['cms-page', slug], queryFn: () => getCmsPage(slug!), enabled: !!slug })
}

export function useBlogPosts() {
  return useQuery({ queryKey: ['blog-posts'], queryFn: listBlogPosts })
}

export function useBlogPost(slug: string | undefined) {
  return useQuery({ queryKey: ['blog-post', slug], queryFn: () => getBlogPost(slug!), enabled: !!slug })
}

export function useFaqItems() {
  return useQuery({ queryKey: ['faq'], queryFn: listFaqItems })
}
