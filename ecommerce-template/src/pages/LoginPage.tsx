import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { loginSchema, type LoginFormValues } from '@/lib/validation'
import { Container } from '@/components/layout/Container'
import { Seo } from '@/components/seo/Seo'
import { Input } from '@/components/ui/Input'
import { Label } from '@/components/ui/Label'
import { Button } from '@/components/ui/Button'
import { Separator } from '@/components/ui/Separator'
import { useAuthStore } from '@/stores/authStore'

export default function LoginPage() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const login = useAuthStore(s => s.login)
  const [error, setError] = useState<string | null>(null)
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<LoginFormValues>({ resolver: zodResolver(loginSchema) })

  return (
    <>
      <Seo title="Log in" />
      <Container className="flex max-w-md flex-col py-16">
        <h1 className="mb-1 text-2xl font-semibold tracking-tight">Log in</h1>
        <p className="mb-6 text-sm text-muted-foreground">Demo account: morgan@example.com / demo-password</p>

        <form
          className="space-y-4"
          noValidate
          onSubmit={handleSubmit(async (values) => {
            setError(null)
            try {
              await login(values.email, values.password)
              navigate(params.get('next') ?? '/account')
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Could not log in.')
            }
          })}
        >
          <div>
            <Label htmlFor="login-email">Email</Label>
            <Input id="login-email" type="email" autoComplete="email" className="mt-1.5" {...register('email')} />
            {errors.email && <p className="mt-1 text-xs text-destructive">{errors.email.message}</p>}
          </div>
          <div>
            <div className="flex items-center justify-between">
              <Label htmlFor="login-password">Password</Label>
              <Link to="/forgot-password" className="text-xs text-primary hover:underline">Forgot password?</Link>
            </div>
            <Input id="login-password" type="password" autoComplete="current-password" className="mt-1.5" {...register('password')} />
            {errors.password && <p className="mt-1 text-xs text-destructive">{errors.password.message}</p>}
          </div>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <Button type="submit" className="w-full" disabled={isSubmitting}>{isSubmitting ? 'Logging in…' : 'Log in'}</Button>
        </form>

        <div className="my-6 flex items-center gap-3"><Separator className="flex-1" /><span className="text-xs text-muted-foreground">or</span><Separator className="flex-1" /></div>

        <div className="space-y-2">
          <Button variant="outline" className="w-full" onClick={() => setError('Social login is a UI placeholder in this demo -- no real provider is connected.')}>Continue with Google</Button>
          <Button variant="outline" className="w-full" onClick={() => setError('Social login is a UI placeholder in this demo -- no real provider is connected.')}>Continue with Apple</Button>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Don't have an account? <Link to="/register" className="text-primary hover:underline">Sign up</Link>
        </p>
      </Container>
    </>
  )
}
