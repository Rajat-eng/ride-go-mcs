import { z } from 'zod';

// ─── Auth request payloads ────────────────────────────────────────────────────

export const LoginPayloadSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
});

export const SignupPayloadSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  name: z.string().min(1, 'Name is required'),
  phoneNumber: z.string().optional(),
  role: z.enum(['rider', 'driver']),
});

// ─── Auth user ────────────────────────────────────────────────────────────────

export const AuthUserSchema = z.object({
  id: z.string().min(1),
  email: z.string().email(),
  name: z.string().min(1),
  phoneNumber: z.string().default(''),
  role: z.enum(['rider', 'driver']),
});

// ─── Auth response ────────────────────────────────────────────────────────────

export const AuthResponseDataSchema = z.object({
  accessToken: z.string().min(1),
  user: AuthUserSchema,
});

export const AuthResponseSchema = z.object({
  data: AuthResponseDataSchema,
});

// ─── Refresh response ─────────────────────────────────────────────────────────

export const RefreshResponseSchema = z.object({
  data: z.object({
    accessToken: z.string().min(1),
    user: AuthUserSchema,
  }),
});

// ─── Exported types ───────────────────────────────────────────────────────────

export type AuthUser = z.infer<typeof AuthUserSchema>;
