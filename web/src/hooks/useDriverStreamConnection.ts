import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { CarPackageSlug } from '../types';
import { TripEvents, ClientWsMessage, BackendEndpoints } from '../contracts';
import { parseWsMessage, validateClientMessage } from '../lib/ws/parseWsMessage';
import { useAppDispatch } from '../store/store';
import {
  setOwnerUserID,
  setDriver,
  setRequestedTrip,
  setTripStatus,
  addChatMessage,
  completeTrip,
  resetTrip,
} from '../store/slices/driverSlice';
import { addError } from '../store/slices/uiSlice';

interface SendOptions {
  reportNotReady?: boolean;
  queueIfNotReady?: boolean;
}

const RECONNECT_BASE_DELAY_MS = 1000;
const RECONNECT_MAX_DELAY_MS = 15000;
const MAX_QUEUED_MESSAGES = 100;

interface useDriverConnectionProps {
  userID: string;
  accessToken: string;
  packageSlug: CarPackageSlug;
}

export const useDriverStreamConnection = ({
  userID,
  accessToken,
  packageSlug,
}: useDriverConnectionProps) => {
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

  useEffect(() => {
    if (!userID || !accessToken || !packageSlug) return;

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

      const websocket = new WebSocket(
        `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(accessToken)}&packageSlug=${encodeURIComponent(packageSlug)}`,
      );
      wsRef.current = websocket;

      websocket.onopen = () => {
        reconnectAttemptsRef.current = 0;
        clearReconnectTimer();
        const activeTripID = activeTripIDRef.current;
        if (activeTripID) {
          sendValidated(websocket, { type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${activeTripID}` } });
          sendValidated(websocket, { type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${activeTripID}:chat` } });
        }
        flushQueuedMessages(websocket);
      };

      websocket.onmessage = (event: MessageEvent<string>) => {
        const result = parseWsMessage(event.data);

        if (!result.ok) {
          emitError(result.error);
          return;
        }

        const message = result.message;

        switch (message.type) {
          case TripEvents.DriverTripRequest: {
            const trip = message.data;
            dispatch(setRequestedTrip(trip));
            dispatch(setTripStatus(message.type));
            if (trip?.id) {
              activeTripIDRef.current = trip.id;
              sendValidated(websocket, { type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${trip.id}` } });
            }
            break;
          }
          case TripEvents.DriverRegister:
            dispatch(setDriver(message.data));
            break;
          case TripEvents.ChatMessageReceived:
            dispatch(addChatMessage(message.data));
            break;
          case TripEvents.Cancelled: {
            const cancelledTripID = message.data?.tripID;
            if (cancelledTripID) {
              sendValidated(websocket, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}` } });
              sendValidated(websocket, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}:chat` } });
              if (activeTripIDRef.current === cancelledTripID) {
                activeTripIDRef.current = null;
              }
            }
            dispatch(resetTrip());
            dispatch(setTripStatus(TripEvents.Cancelled));
            break;
          }
          case TripEvents.Completed: {
            const completedTripID = message.data?.tripID ?? activeTripIDRef.current;
            if (completedTripID) {
              sendValidated(websocket, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${completedTripID}` } });
              sendValidated(websocket, { type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${completedTripID}:chat` } });
            }
            activeTripIDRef.current = null;
            dispatch(completeTrip());
            break;
          }
        }
      };

      websocket.onclose = (e: CloseEvent) => {
        if (wsRef.current === websocket) {
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

      websocket.onerror = () => { /* onclose fires next with the close code — handled there */ };
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
  }, [userID, accessToken, packageSlug, clearReconnectTimer, dispatch, emitError, flushQueuedMessages, sendValidated]);

  // stable across renders — reads from ref so it always uses the live socket
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
