import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from 'react-router-dom'
import { registerSchema, type RegisterFormValues } from '@/lib/validation'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { useAuthStore } from '@/stores/authStore'

export default function RegisterPage() {
  const navigate = useNavigate()
  const registerUser = useAuthStore(s => s.register)
  const [error, setError] = useState<string | null>(null)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<RegisterFormValues>({ resolver: zodResolver(registerSchema) })

  return (
    <>
      <Seo title="Create an account" />
      <Container className="max-w-md py-16">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">Create an account</h1>
        <p className="mb-6 text-sm text-muted-foreground">Or check out as a guest any time -- an account just saves your details for next time.</p>

        <form
          className="space-y-4"
          noValidate
          onSubmit={handleSubmit(async (values) => {
            setError(null)
            try {
              await registerUser(values)
              navigate('/account')
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Could not create your account.')
            }
          })}
        >
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="register-first-name">First name</Label>
              <Input id="register-first-name" autoComplete="given-name" className="mt-1.5" {...register('firstName')} />
              {errors.firstName && <p className="mt-1 text-xs text-destructive">{errors.firstName.message}</p>}
            </div>
            <div>
              <Label htmlFor="register-last-name">Last name</Label>
              <Input id="register-last-name" autoComplete="family-name" className="mt-1.5" {...register('lastName')} />
              {errors.lastName && <p className="mt-1 text-xs text-destructive">{errors.lastName.message}</p>}
            </div>
          </div>
          <div>
            <Label htmlFor="register-email">Email</Label>
            <Input id="register-email" type="email" autoComplete="email" className="mt-1.5" {...register('email')} />
            {errors.email && <p className="mt-1 text-xs text-destructive">{errors.email.message}</p>}
          </div>
          <div>
            <Label htmlFor="register-password">Password</Label>
            <Input id="register-password" type="password" autoComplete="new-password" className="mt-1.5" {...register('password')} />
            {errors.password && <p className="mt-1 text-xs text-destructive">{errors.password.message}</p>}
          </div>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full" disabled={isSubmitting}>{isSubmitting ? 'Creating account…' : 'Create account'}</Button>
        </form>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Already have an account? <Link to="/login" className="text-primary hover:underline">Log in</Link>
        </p>
      </Container>
    </>
  )
}
