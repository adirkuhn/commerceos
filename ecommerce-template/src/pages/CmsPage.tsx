import { useParams } from 'react-router-dom'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { LoadingState, NotFound } from '@/components/ui/States'
import { SectionList } from '@/components/sections/SectionRenderer'
import { useCmsPage, useFaqItems } from '@/features/content/hooks'
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from '@/components/ui/Accordion'

export default function CmsPage() {
  const { slug } = useParams<{ slug: string }>()
  const { data: page, isLoading } = useCmsPage(slug)
  const { data: faqItems = [] } = useFaqItems()

  if (isLoading) return <LoadingState />
  if (!page) return <NotFound title="Page not found" />

  return (
    <>
      <Seo title={page.title} description={page.metaDescription} />
      <Container className="max-w-3xl space-y-16 py-10">
        <SectionList sections={page.sections} />
        {slug === 'shipping-returns' && (
          <section>
            <h2 className="mb-4 text-xl font-semibold">Frequently asked questions</h2>
            <Accordion type="single" collapsible>
              {faqItems.map((item, i) => (
                <AccordionItem key={i} value={`faq-${i}`}>
                  <AccordionTrigger>{item.question}</AccordionTrigger>
                  <AccordionContent>{item.answer}</AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </section>
        )}
      </Container>
    </>
  )
}
