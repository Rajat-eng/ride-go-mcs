/**
 * Schemas for every client → server WebSocket message.
 *
 * Validating outgoing payloads catches shape mistakes at the send site
 * before they hit the wire, giving clearer errors than a silent drop.
 */
import { z } from 'zod';
import { TripEvents } from '../../contracts';
import { CoordinateSchema } from './domain.schemas';

// ─── Topic control (legacy + room model) ─────────────────────────────────────

export const WsTopicSubscribeSchema = z.object({
  type: z.literal(TripEvents.WsTopicSubscribe),
  data: z.object({ topic: z.string().min(1) }),
});

export const WsTopicUnsubscribeSchema = z.object({
  type: z.literal(TripEvents.WsTopicUnsubscribe),
  data: z.object({ topic: z.string().min(1) }),
});

// ─── Driver → server ─────────────────────────────────────────────────────────

export const DriverLocationMessageSchema = z.object({
  type: z.literal(TripEvents.DriverLocation),
  data: z.object({
    location: CoordinateSchema,
    geohash: z.string().min(1),
  }),
});

export const DriverTripAcceptSchema = z.object({
  type: z.literal(TripEvents.DriverTripAccept),
  data: z.object({
    tripID: z.string().min(1),
    riderID: z.string().min(1),
  }),
});

export const DriverTripDeclineSchema = z.object({
  type: z.literal(TripEvents.DriverTripDecline),
  data: z.object({
    tripID: z.string().min(1),
    riderID: z.string().min(1),
  }),
});

// ─── Rider → server ───────────────────────────────────────────────────────────

export const ChatMessageSendSchema = z.object({
  type: z.literal(TripEvents.ChatMessageSend),
  data: z.object({
    tripID: z.string().min(1),
    text: z.string().min(1),
    messageID: z.string().optional(),
  }),
});

export const TripCancelClientSchema = z.object({
  type: z.literal(TripEvents.TripCmdCancel),
  data: z.object({ tripID: z.string().min(1) }),
});

// ─── Full client message discriminated union ─────────────────────────────────
// All members have a unique z.literal() on `type` — O(1) dispatch.

export const ClientWsMessageSchema = z.discriminatedUnion('type', [
  WsTopicSubscribeSchema,
  WsTopicUnsubscribeSchema,
  DriverLocationMessageSchema,
  DriverTripAcceptSchema,
  DriverTripDeclineSchema,
  ChatMessageSendSchema,
  TripCancelClientSchema,
]);

export type ValidatedClientWsMessage = z.infer<typeof ClientWsMessageSchema>;
