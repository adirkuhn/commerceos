import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { NotFound } from '@/components/ui/States'

export default function NotFoundPage() {
  return (
    <>
      <Seo title="Page not found" />
      <Container><NotFound /></Container>
    </>
  )
}
