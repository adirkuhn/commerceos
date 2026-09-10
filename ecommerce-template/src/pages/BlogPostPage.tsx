import { useParams } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { Breadcrumbs } from '@/components/layout/Breadcrumbs'
import { Seo } from '@/components/seo/Seo'
import { LoadingState, NotFound } from '@/components/ui/States'
import { useBlogPost } from '@/features/content/hooks'
import { formatDate } from '@/lib/utils'

export default function BlogPostPage() {
  const { slug } = useParams<{ slug: string }>()
  const { data: post, isLoading } = useBlogPost(slug)

  if (isLoading) return <LoadingState />
  if (!post) return <NotFound title="Article not found" />

  return (
    <>
      <Seo title={post.title} description={post.excerpt} image={post.coverImageUrl} />
      <Container className="max-w-2xl pb-16">
        <Breadcrumbs items={[{ label: 'Journal', href: '/blog' }, { label: post.title }]} />
        <h1 className="mb-2 mt-4 text-3xl font-semibold tracking-tight">{post.title}</h1>
        <p className="mb-6 text-sm text-muted-foreground">{formatDate(post.publishedAt)} · {post.author}</p>
        <img src={post.coverImageUrl} alt="" className="mb-8 aspect-[16/9] w-full rounded-[var(--radius)] object-cover" />
        <div
          className="space-y-4 [&_p]:text-muted-foreground [&_p]:leading-relaxed"
          dangerouslySetInnerHTML={{ __html: post.bodyHtml }}
        />
      </Container>
    </>
  )
}
