import { useProductQuestions } from '@/features/reviews/hooks'
import { formatDate } from '@/lib/utils'
import { LoadingState } from '@/components/ui/States'

export function QASection({ productId }: { productId: string }) {
  const { data: questions = [], isLoading } = useProductQuestions(productId)

  if (isLoading) return <LoadingState />
  if (questions.length === 0) return <p className="py-4 text-sm text-muted-foreground">No questions yet for this product.</p>

  return (
    <ul className="space-y-5">
      {questions.map(q => (
        <li key={q.id} className="rounded-[var(--radius)] border border-border p-4">
          <p className="text-sm font-medium">Q: {q.question}</p>
          <p className="text-xs text-muted-foreground">{q.author} · {formatDate(q.createdAt)}</p>
          {q.answer && (
            <div className="mt-3 border-l-2 border-primary pl-3">
              <p className="text-sm text-muted-foreground">A: {q.answer}</p>
              {q.answeredBy && <p className="text-xs text-muted-foreground">— {q.answeredBy}</p>}
            </div>
          )}
        </li>
      ))}
    </ul>
  )
}
