import { z } from 'zod'

export const addressSchema = z.object({
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  company: z.string().optional(),
  line1: z.string().min(1, 'Street address is required'),
  line2: z.string().optional(),
  city: z.string().min(1, 'City is required'),
  province: z.string().optional(),
  postalCode: z.string().min(1, 'Postal code is required'),
  country: z.string().min(2, 'Country is required'),
  phone: z.string().optional(),
  vatNumber: z.string().optional(),
})
export type AddressFormValues = z.infer<typeof addressSchema>

export const loginSchema = z.object({
  email: z.string().email('Enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
})
export type LoginFormValues = z.infer<typeof loginSchema>

export const registerSchema = z.object({
  firstName: z.string().min(1, 'First name is required'),
  lastName: z.string().min(1, 'Last name is required'),
  email: z.string().email('Enter a valid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})
export type RegisterFormValues = z.infer<typeof registerSchema>

export const forgotPasswordSchema = z.object({ email: z.string().email('Enter a valid email address') })
export type ForgotPasswordFormValues = z.infer<typeof forgotPasswordSchema>

export const resetPasswordSchema = z.object({
  password: z.string().min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string(),
}).refine(data => data.password === data.confirmPassword, { message: 'Passwords must match', path: ['confirmPassword'] })
export type ResetPasswordFormValues = z.infer<typeof resetPasswordSchema>

export const checkoutCustomerSchema = z.object({ email: z.string().email('Enter a valid email address'), phone: z.string().optional() })
export type CheckoutCustomerFormValues = z.infer<typeof checkoutCustomerSchema>

export const reviewSchema = z.object({
  rating: z.number().min(1).max(5),
  title: z.string().min(1, 'Give your review a title'),
  body: z.string().min(10, 'Reviews need at least 10 characters'),
})
export type ReviewFormValues = z.infer<typeof reviewSchema>
