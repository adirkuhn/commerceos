import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { forgotPasswordSchema, type ForgotPasswordFormValues } from '@/lib/validation'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { authService } from '@/services/authService'

export default function ForgotPasswordPage() {
  const [sent, setSent] = useState(false)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<ForgotPasswordFormValues>({ resolver: zodResolver(forgotPasswordSchema) })

  return (
    <>
      <Seo title="Forgot password" />
      <Container className="max-w-md py-16">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">Reset your password</h1>
        <p className="mb-6 text-sm text-muted-foreground">We'll email you a link to reset it.</p>

        {sent ? (
          <p className="rounded-[var(--radius)] bg-accent p-4 text-sm text-accent-foreground">
            If an account exists for that email, a reset link is on its way. In this demo, use{' '}
            <Link to="/reset-password?token=demo-token" className="underline">this link</Link> to continue.
          </p>
        ) : (
          <form
            className="space-y-4"
            noValidate
            onSubmit={handleSubmit(async (values) => { await authService.requestPasswordReset(values.email); setSent(true) })}
          >
            <div>
              <Label htmlFor="forgot-email">Email</Label>
              <Input id="forgot-email" type="email" autoComplete="email" className="mt-1.5" {...register('email')} />
              {errors.email && <p className="mt-1 text-xs text-destructive">{errors.email.message}</p>}
            </div>
            <Button type="submit" className="w-full" disabled={isSubmitting}>{isSubmitting ? 'Sending…' : 'Send reset link'}</Button>
          </form>
        )}
      </Container>
    </>
  )
}
