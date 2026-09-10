import { contentRepository } from '@/repositories'

export const getCmsPage = (slug: string) => contentRepository.getPage(slug)
export const listBlogPosts = () => contentRepository.listBlogPosts()
export const getBlogPost = (slug: string) => contentRepository.getBlogPost(slug)
export const listFaqItems = () => contentRepository.faq()
