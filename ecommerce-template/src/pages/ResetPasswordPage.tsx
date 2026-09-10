import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { resetPasswordSchema, type ResetPasswordFormValues } from '@/lib/validation'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { authService } from '@/services/authService'
import { toast } from '@/components/ui/toast'

export default function ResetPasswordPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [error, setError] = useState<string | null>(null)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<ResetPasswordFormValues>({ resolver: zodResolver(resetPasswordSchema) })

  return (
    <>
      <Seo title="Set a new password" />
      <Container className="max-w-md py-16">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">Set a new password</h1>
        <form
          className="mt-6 space-y-4"
          noValidate
          onSubmit={handleSubmit(async (values) => {
            setError(null)
            try {
              await authService.resetPassword(token, values.password)
              toast({ title: 'Password updated', variant: 'success' })
              navigate('/login')
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Could not reset your password.')
            }
          })}
        >
          <div>
            <Label htmlFor="reset-password">New password</Label>
            <Input id="reset-password" type="password" autoComplete="new-password" className="mt-1.5" {...register('password')} />
            {errors.password && <p className="mt-1 text-xs text-destructive">{errors.password.message}</p>}
          </div>
          <div>
            <Label htmlFor="reset-confirm">Confirm password</Label>
            <Input id="reset-confirm" type="password" autoComplete="new-password" className="mt-1.5" {...register('confirmPassword')} />
            {errors.confirmPassword && <p className="mt-1 text-xs text-destructive">{errors.confirmPassword.message}</p>}
          </div>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full" disabled={isSubmitting}>{isSubmitting ? 'Saving…' : 'Reset password'}</Button>
        </form>
      </Container>
    </>
  )
}
