import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { TripEvents, BackendEndpoints, ClientWsMessage } from '../contracts';
import { parseWsMessage, validateClientMessage } from '../lib/ws/parseWsMessage';
import { useAppDispatch } from '../store/store';
import {
  setOwnerUserID,
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setAssignedDriverLocation,
  addChatMessage,
  completeTrip,
  resetTrip,
} from '../store/slices/riderSlice';
import { addError } from '../store/slices/uiSlice';

interface SendOptions {
  reportNotReady?: boolean;
  queueIfNotReady?: boolean;
}

const RECONNECT_BASE_DELAY_MS = 1000;
const RECONNECT_MAX_DELAY_MS = 15000;
const MAX_QUEUED_MESSAGES = 100;

export function useRiderStreamConnection(userID: string, accessToken: string) {
  const dispatch = useAppDispatch();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const shouldReconnectRef = useRef(false);
  const outboundQueueRef = useRef<ClientWsMessage[]>([]);
  const activeTripIDRef = useRef<string | null>(null);

  const emitError = useCallback((message: string) => {
    dispatch(addError({ message, timestamp: Date.now() }));
  }, [dispatch]);

  const clearReconnectTimer = useCallback(() => {
    if (!reconnectTimerRef.current) return;
    clearTimeout(reconnectTimerRef.current);
    reconnectTimerRef.current = null;
  }, []);

  const enqueueMessage = useCallback((message: ClientWsMessage) => {
    if (outboundQueueRef.current.length >= MAX_QUEUED_MESSAGES) {
      outboundQueueRef.current.shift();
    }
    outboundQueueRef.current.push(message);
  }, []);

  const sendValidated = useCallback((ws: WebSocket, message: ClientWsMessage, options?: SendOptions): boolean => {
    const validation = validateClientMessage(message);
    if (!validation.ok) {
      emitError(validation.error);
      return false;
    }

    if (ws.readyState !== WebSocket.OPEN) {
      if (options?.queueIfNotReady !== false) {
        enqueueMessage(validation.payload);
        return true;
      }
      if (options?.reportNotReady) {
        emitError('Connection is not ready. Please wait and try again.');
      }
      return false;
    }

    ws.send(JSON.stringify(validation.payload));
    return true;
  }, [emitError, enqueueMessage]);

  const flushQueuedMessages = useCallback((ws: WebSocket) => {
    if (ws.readyState !== WebSocket.OPEN || outboundQueueRef.current.length === 0) {
      return;
    }

    const pending = outboundQueueRef.current;
    outboundQueueRef.current = [];

    for (const queued of pending) {
      const sent = sendValidated(ws, queued, { queueIfNotReady: false });
      if (!sent) {
        enqueueMessage(queued);
      }
    }
  }, [enqueueMessage, sendValidated]);

  const joinTripRooms = useCallback((ws: WebSocket, tripID?: string) => {
    if (!tripID) return;
    activeTripIDRef.current = tripID;
    sendValidated(ws, { type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}` } });
    sendValidated(ws, { type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}:chat` } });
  }, [sendValidated]);

  useEffect(() => {
    if (!userID || !accessToken) return;

    dispatch(setOwnerUserID(userID));
    shouldReconnectRef.current = true;

    const scheduleReconnect = () => {
      if (!shouldReconnectRef.current || reconnectTimerRef.current) {
        return;
      }

      reconnectAttemptsRef.current += 1;
      const exponentialDelay = Math.min(
        RECONNECT_MAX_DELAY_MS,
        RECONNECT_BASE_DELAY_MS * (2 ** (reconnectAttemptsRef.current - 1)),
      );
      const jitterMs = Math.floor(Math.random() * 250);
      const delayMs = exponentialDelay + jitterMs;

      reconnectTimerRef.current = setTimeout(() => {
        reconnectTimerRef.current = null;
        connect();
      }, delayMs);
    };

    const connect = () => {
      if (!shouldReconnectRef.current) {
        return;
      }

      const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?token=${encodeURIComponent(accessToken)}`);
      wsRef.current = ws;

      ws.onopen = () => {
        reconnectAttemptsRef.current = 0;
        clearReconnectTimer();
        const activeTripID = activeTripIDRef.current;
        if (activeTripID) {
          joinTripRooms(ws, activeTripID);
        }
        flushQueuedMessages(ws);
      };

      ws.onmessage = (event: MessageEvent<string>) => {
        const result = parseWsMessage(event.data);

        if (!result.ok) {
          emitError(result.error);
          return;
        }

        const message = result.message;

        switch (message.type) {
          case TripEvents.DriverLocation:
            dispatch(setDrivers(message.data));
            break;
          case TripEvents.DriverEventLocation:
            dispatch(setAssignedDriverLocation(message.data));
            break;
          case TripEvents.PaymentSessionCreated:
            dispatch(setPaymentSession(message.data));
            dispatch(setTripStatus(message.type));
            joinTripRooms(ws, message.data.tripID);
            break;
          case TripEvents.DriverAssigned: {
            const trip = message.data;
            dispatch(setAssignedDriver(trip.driver ? {
              id: trip.driver.id,
              name: trip.driver.name,
              location: { latitude: 0, longitude: 0 },
              geohash: '', // once joined to trip room, will receive location updates to fill this in
              profilePicture: '',
              carPlate: '',
            } : null));
            dispatch(setTripStatus(message.type));
            joinTripRooms(ws, trip.id);
            break;
          }
          case TripEvents.Created:
            dispatch(setTripStatus(message.type));
            joinTripRooms(ws, message.data.id);
            break;
          case TripEvents.NoDriversFound:
            dispatch(setTripStatus(message.type));
            break;
          case TripEvents.ChatMessageReceived:
            dispatch(addChatMessage(message.data));
            break;
          case TripEvents.Cancelled: {
            dispatch(resetTrip());
            dispatch(setTripStatus(TripEvents.Cancelled));
            const cancelledTripID = message.data?.tripID;
            if (cancelledTripID) {
              sendValidated(ws, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}` } });
              sendValidated(ws, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}:chat` } });
              if (activeTripIDRef.current === cancelledTripID) {
                activeTripIDRef.current = null;
              }
            }
            break;
          }
          case TripEvents.Completed: {
            const completedTripID = message.data?.tripID ?? activeTripIDRef.current;
            if (completedTripID) {
              sendValidated(ws, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${completedTripID}` } });
              sendValidated(ws, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${completedTripID}:chat` } });
            }
            activeTripIDRef.current = null;
            dispatch(completeTrip());
            break;
          }
        }
      };

      ws.onclose = (e: CloseEvent) => {
        if (wsRef.current === ws) {
          wsRef.current = null;
        }
        if (!shouldReconnectRef.current) {
          return;
        }
        if (e.code !== 1000 && e.code !== 1001) {
          emitError(e.reason || 'Live connection interrupted. Reconnecting...');
        }
        scheduleReconnect();
      };

      ws.onerror = () => { /* onclose fires next with the close code — handled there */ };
    };

    connect();

    return () => {
      shouldReconnectRef.current = false;
      clearReconnectTimer();
      reconnectAttemptsRef.current = 0;
      outboundQueueRef.current = [];
      activeTripIDRef.current = null;

      const ws = wsRef.current;
      if (ws && ws.readyState < WebSocket.CLOSING) {
        ws.close(1000, 'Component unmounted');
      }
      wsRef.current = null;
    };
  }, [userID, accessToken, clearReconnectTimer, dispatch, emitError, flushQueuedMessages, joinTripRooms, sendValidated]);

  const sendMessage = useCallback((message: ClientWsMessage, options?: SendOptions): boolean => {
    const ws = wsRef.current;
    if (!ws) {
      const validation = validateClientMessage(message);
      if (!validation.ok) {
        emitError(validation.error);
        return false;
      }
      if (options?.queueIfNotReady !== false) {
        enqueueMessage(validation.payload);
        return true;
      }
      if (options?.reportNotReady) {
        emitError('Connection is not ready. Please wait and try again.');
      }
      return false;
    }
    return sendValidated(ws, message, options);
  }, [emitError, enqueueMessage, sendValidated]);

  return { sendMessage };
}
