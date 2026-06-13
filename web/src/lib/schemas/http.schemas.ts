/**
 * Schemas for HTTP request payloads and API response envelopes.
 */
import { z } from 'zod';
import { RouteFareSchema, RouteSchema } from './domain.schemas';

// ─── Generic API envelope ─────────────────────────────────────────────────────

/**
 * Wraps any data shape in the standard { data: T } response envelope
 * used by the API gateway.
 */
export function apiEnvelope<T extends z.ZodTypeAny>(dataSchema: T) {
  return z.object({ data: dataSchema });
}

// ─── Trip preview ─────────────────────────────────────────────────────────────

export const TripPreviewResponseSchema = z.object({
  route: RouteSchema,
  rideFares: z.array(RouteFareSchema),
});

export const TripPreviewApiResponseSchema = apiEnvelope(TripPreviewResponseSchema);

// ─── Trip start ───────────────────────────────────────────────────────────────

export const StartTripResponseSchema = z.object({
  tripID: z.string().min(1),
});

// ─── Stripe config ────────────────────────────────────────────────────────────

export const StripeConfigResponseSchema = z.object({
  publishableKey: z.string().default(''),
});

// ─── Exported types ───────────────────────────────────────────────────────────

export type TripPreviewResponse = z.infer<typeof TripPreviewResponseSchema>;
export type StartTripResponse = z.infer<typeof StartTripResponseSchema>;
