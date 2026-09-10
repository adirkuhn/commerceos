import { Link } from 'react-router-dom'
import { Heart, Share2 } from 'lucide-react'
import { Container } from '@/components/layout/Container'
import { PageHeader } from '@/components/layout/PageHeader'
import { Seo } from '@/components/seo/Seo'
import { Button } from '@/components/ui/Button'
import { EmptyState, LoadingState } from '@/components/ui/States'
import { ProductGrid } from '@/components/ecommerce/ProductGrid'
import { useWishlistProducts } from '@/features/wishlist/hooks'
import { toast } from '@/components/ui/toast'

export default function WishlistPage() {
  const { data: products, isLoading } = useWishlistProducts()

  return (
    <>
      <Seo title="Wishlist" />
      <Container className="pb-16">
        <PageHeader
          title="Wishlist"
          action={
            <Button
              variant="outline" size="sm"
              onClick={async () => {
                await navigator.clipboard.writeText(window.location.href).catch(() => {})
                toast({ title: 'Wishlist link copied' })
              }}
            >
              <Share2 className="mr-2 h-4 w-4" /> Share wishlist
            </Button>
          }
        />
        {isLoading && <LoadingState />}
        {!isLoading && (!products || products.length === 0) && (
          <EmptyState
            icon={<Heart className="h-8 w-8" />}
            title="Your wishlist is empty"
            description="Save items you love to find them here later."
            action={<Button asChild className="mt-2"><Link to="/">Browse products</Link></Button>}
          />
        )}
        {products && products.length > 0 && <ProductGrid products={products} />}
      </Container>
    </>
  )
}
