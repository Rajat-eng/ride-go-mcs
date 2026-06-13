import { z } from 'zod';
import { CarPackageSlug } from '../../types';

// ─── Primitives ───────────────────────────────────────────────────────────────

export const CoordinateSchema = z.object({
  latitude: z.number(),
  longitude: z.number(),
});

// ─── Driver ───────────────────────────────────────────────────────────────────

export const DriverSchema = z.object({
  id: z.string().min(1),
  location: CoordinateSchema,
  geohash: z.string().min(1),
  name: z.string().min(1),
  profilePicture: z.string().catch(''),
  carPlate: z.string().min(1),
});

export type ParsedDriver = z.infer<typeof DriverSchema>;

// ─── Route ────────────────────────────────────────────────────────────────────

export const RouteGeometrySchema = z.object({
  coordinates: z.array(CoordinateSchema),
});

export const RouteSchema = z.object({
  geometry: z.array(RouteGeometrySchema),
  duration: z.number(),
  distance: z.number(),
});

// ─── Fare ─────────────────────────────────────────────────────────────────────

export const CarPackageSlugSchema = z.nativeEnum(CarPackageSlug);

export const RouteFareSchema = z.object({
  id: z.string().default(''),
  packageSlug: CarPackageSlugSchema,
  basePrice: z.number().optional().default(0),
  totalPriceInCents: z.number().optional(),
  // expiresAt is only present on fully persisted fares, not preview fares.
  expiresAt: z.coerce.date().optional(),
  route: RouteSchema.optional(),
});

// ─── Trip ─────────────────────────────────────────────────────────────────────

export const TripDriverSchema = z.object({
  id: z.string().optional().default(''),
  name: z.string().optional().default(''),
});

export const TripSchema = z.object({
  id: z.string().min(1),
  userID: z.string().min(1),
  status: z.string(),
  selectedFare: RouteFareSchema.optional(),
  route: RouteSchema.optional(),
  driver: TripDriverSchema.optional(),
});

// ─── Payment ──────────────────────────────────────────────────────────────────

export const PaymentSessionDataSchema = z.object({
  tripID: z.string().min(1),
  sessionID: z.string().min(1),
  amount: z.number(),
  currency: z.string().min(1),
});

export type PaymentSessionData = z.infer<typeof PaymentSessionDataSchema>;

// ─── Chat ─────────────────────────────────────────────────────────────────────

export const ChatMessageDataSchema = z.object({
  tripID: z.string().min(1),
  senderID: z.string().min(1),
  text: z.string().min(1),
  sentAt: z.number(),
  messageID: z.string().optional(),
});

export type ChatMessageData = z.infer<typeof ChatMessageDataSchema>;
