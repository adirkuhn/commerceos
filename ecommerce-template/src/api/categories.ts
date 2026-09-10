import { categoryRepository } from '@/repositories'

export const listCategories = () => categoryRepository.list()
export const getCategoryBySlug = (slug: string) => categoryRepository.getBySlug(slug)
export const listBrands = () => categoryRepository.brands()
