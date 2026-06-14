/**
 * Schemas for every server → client WebSocket message.
 *
 * Each event is a discriminated z.object keyed on `type`.
 * The union is tagged so TypeScript narrows cleanly after
 * a safeParse / discriminatedUnion parse.
 */
import { z } from 'zod';
import { TripEvents } from '../../contracts';
import {
  CoordinateSchema,
  DriverSchema,
  ChatMessageDataSchema,
  PaymentSessionDataSchema,
  TripSchema,
} from './domain.schemas';

// ─── Individual event schemas ─────────────────────────────────────────────────

export const DriverLocationEventSchema = z.object({
  type: z.literal(TripEvents.DriverLocation),
  topic: z.string().optional(),
  data: z.array(DriverSchema),
});

export const DriverEventLocationSchema = z.object({
  type: z.literal(TripEvents.DriverEventLocation),
  topic: z.string().optional(),
  data: CoordinateSchema,
});

export const DriverRegisterSchema = z.object({
  type: z.literal(TripEvents.DriverRegister),
  topic: z.string().optional(),
  data: DriverSchema,
});

export const DriverTripRequestSchema = z.object({
  type: z.literal(TripEvents.DriverTripRequest),
  topic: z.string().optional(),
  // Gateway may nest under `trip` or send flat — handle both
  data: z.union([
    TripSchema,
    z.object({ trip: TripSchema }).transform((d) => d.trip),
  ]),
});

export const DriverAssignedSchema = z.object({
  type: z.literal(TripEvents.DriverAssigned),
  topic: z.string().optional(),
  data: TripSchema,
});

export const TripCreatedSchema = z.object({
  type: z.literal(TripEvents.Created),
  topic: z.string().optional(),
  // May arrive as { trip: Trip, pickupLat, pickupLng } or flat Trip
  data: z.union([
    TripSchema,
    z
      .object({
        trip: TripSchema,
        pickupLat: z.number().optional(),
        pickupLng: z.number().optional(),
      })
      .transform((d) => d.trip),
  ]),
});

export const NoDriversFoundSchema = z.object({
  type: z.literal(TripEvents.NoDriversFound),
  topic: z.string().optional(),
  data: z.unknown().optional(),
});

export const TripCancelledSchema = z.object({
  type: z.literal(TripEvents.Cancelled),
  topic: z.string().optional(),
  // data is optional at envelope level but tripID must be a non-empty string when present
  data: z.object({ tripID: z.string().min(1) }).optional(),
});

export const TripCompletedSchema = z.object({
	 type: z.literal(TripEvents.Completed),
	 topic: z.string().optional(),
	 data: z.object({ tripID: z.string().min(1) }).optional(),
});

export const PaymentSessionCreatedSchema = z.object({
  type: z.literal(TripEvents.PaymentSessionCreated),
  topic: z.string().optional(),
  data: PaymentSessionDataSchema,
});

export const ChatMessageReceivedSchema = z.object({
  type: z.literal(TripEvents.ChatMessageReceived),
  topic: z.string().optional(),
  data: ChatMessageDataSchema,
});

// ─── Discriminated union ──────────────────────────────────────────────────────
//
// z.discriminatedUnion gives O(1) lookup on the `type` field and tighter
// narrowing vs a plain z.union.

export const ServerWsMessageSchema = z.discriminatedUnion('type', [
  DriverLocationEventSchema,
  DriverEventLocationSchema,
  DriverRegisterSchema,
  DriverTripRequestSchema,
  DriverAssignedSchema,
  TripCreatedSchema,
  NoDriversFoundSchema,
  TripCancelledSchema,
  TripCompletedSchema,
  PaymentSessionCreatedSchema,
  ChatMessageReceivedSchema,
]);

export type ParsedServerWsMessage = z.infer<typeof ServerWsMessageSchema>;
