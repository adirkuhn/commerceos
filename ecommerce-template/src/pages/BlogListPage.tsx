import { Link } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { PageHeader } from '@/components/layout/PageHeader'
import { Seo } from '@/components/seo/Seo'
import { LoadingState } from '@/components/ui/States'
import { useBlogPosts } from '@/features/content/hooks'
import { formatDate } from '@/lib/utils'

export default function BlogListPage() {
  const { data: posts = [], isLoading } = useBlogPosts()

  return (
    <>
      <Seo title="Journal" description="Field guides and gear notes from Aurora." />
      <Container className="pb-16">
        <PageHeader title="Journal" description="Field guides and gear notes." />
        {isLoading ? <LoadingState /> : (
          <div className="grid gap-8 sm:grid-cols-2">
            {posts.map(post => (
              <Link key={post.id} to={`/blog/${post.slug}`} className="group">
                <img src={post.coverImageUrl} alt="" className="mb-3 aspect-[16/9] w-full rounded-[var(--radius)] object-cover" />
                <p className="text-xs text-muted-foreground">{formatDate(post.publishedAt)} · {post.author}</p>
                <h2 className="mt-1 text-lg font-semibold group-hover:underline">{post.title}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{post.excerpt}</p>
              </Link>
            ))}
          </div>
        )}
      </Container>
    </>
  )
}
