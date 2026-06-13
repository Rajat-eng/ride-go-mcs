import { Coordinate, Driver, Route, RouteFare, Trip } from "./types";


// These are the endpoints the API Gateway must have for the frontend to work correctly
export enum BackendEndpoints {
  PREVIEW_TRIP = "/trip/preview",
  START_TRIP = "/trip/start",
  CANCEL_TRIP = "/trip/cancel",
  WS_DRIVERS = "/drivers",
  WS_RIDERS = "/riders",
}

export enum TripEvents {
  NoDriversFound = "trip.event.no_drivers_found",
  DriverAssigned = "trip.event.driver_assigned",
  Completed = "trip.event.completed",
  Cancelled = "trip.event.cancelled",
  TripCmdCancel = "trip.cmd.cancel",
  Created = "trip.event.created",
  DriverLocation = "driver.cmd.location",
  DriverTripRequest = "driver.cmd.trip_request",
  DriverTripAccept = "driver.cmd.trip_accept",
  DriverTripDecline = "driver.cmd.trip_decline",
  DriverRegister = "driver.cmd.register",
  DriverEventLocation = "driver.event.location",
  PaymentSessionCreated = "payment.event.session_created",
  ChatMessageSend = "chat.message.send",
  ChatMessageReceived = "chat.message.received",
  WsTopicSubscribe = "ws.topic.subscribe",
  WsTopicUnsubscribe = "ws.topic.unsubscribe",
}

// Every server message may carry an optional topic for client-side filtering.
export type ServerWsMessage = (
  | PaymentSessionCreatedRequest
  | DriverAssignedRequest
  | DriverLocationRequest
  | DriverEventLocationRequest
  | ChatMessageReceivedRequest
  | DriverTripRequest
  | DriverRegisterRequest
  | TripCreatedRequest
  | NoDriversFoundRequest
  | TripCancelledRequest
) & { topic?: string };

// Messages sent from the client to the server via the websocket
export type ClientWsMessage =
  | DriverResponseToTripResponse
  | DriverLocationMessage
  | ChatMessageSendRequest
  | WsTopicSubscribeMessage
  | WsTopicUnsubscribeMessage
  | TripCancelRequest

export interface WsTopicSubscribeMessage {
  type: TripEvents.WsTopicSubscribe;
  data: { topic: string };
}

export interface WsTopicUnsubscribeMessage {
  type: TripEvents.WsTopicUnsubscribe;
  data: { topic: string };
}

interface TripCreatedRequest {
  type: TripEvents.Created;
  data: Trip;
}

interface NoDriversFoundRequest {
  type: TripEvents.NoDriversFound;
}

interface TripCancelledRequest {
  type: TripEvents.Cancelled;
  data: {
    tripID: string;
  };
}

interface DriverRegisterRequest {
  type: TripEvents.DriverRegister;
  data: Driver;
}
interface DriverTripRequest {
  type: TripEvents.DriverTripRequest;
  data: Trip;
}

export interface PaymentEventSessionCreatedData {
  tripID: string;
  sessionID: string;
  amount: number;
  currency: string;
}

interface PaymentSessionCreatedRequest {
  type: TripEvents.PaymentSessionCreated;
  data: PaymentEventSessionCreatedData;
}

interface DriverAssignedRequest {
  type: TripEvents.DriverAssigned;
  data: Trip;
}

interface DriverLocationRequest {
  type: TripEvents.DriverLocation;
  data: Driver[];
}

interface DriverEventLocationRequest {
  type: TripEvents.DriverEventLocation;
  data: Coordinate;
}

export interface ChatMessageData {
  tripID: string;
  senderID: string;
  text: string;
  sentAt: number;
  messageID?: string;
}

interface ChatMessageReceivedRequest {
  type: TripEvents.ChatMessageReceived;
  data: ChatMessageData;
}

interface ChatMessageSendRequest {
  type: TripEvents.ChatMessageSend;
  data: {
    tripID: string;
    text: string;
    messageID?: string;
  };
}

interface DriverResponseToTripResponse {
  type: TripEvents.DriverTripAccept | TripEvents.DriverTripDecline;
  data: {
    tripID: string;
    riderID: string;
  };
}

interface DriverLocationMessage {
  type: TripEvents.DriverLocation;
  data: {
    location: Coordinate;
    geohash: string;
  };
}

interface TripCancelRequest {
  type: TripEvents.TripCmdCancel;
  data: {
    tripID: string;
  };
}

export interface HTTPTripPreviewResponse {
  route: Route;
  rideFares: RouteFare[];
}

export interface HTTPTripStartRequestPayload {
  rideFareID: string;
  userID: string;
}

export interface HTTPTripPreviewRequestPayload {
  userID: string;
  pickup: Coordinate;
  destination: Coordinate;
}


