import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { CarPackageSlug } from '../types';
import { ServerWsMessage, TripEvents, isValidWsMessage, ClientWsMessage, BackendEndpoints } from '../contracts';
import { useAppDispatch } from '../store/store';
import {
  setDriver,
  setRequestedTrip,
  setTripStatus,
  addChatMessage,
  resetTrip,
} from '../store/slices/driverSlice';
import { logout } from '../store/slices/authSlice';
import { addError } from '../store/slices/uiSlice';

function isTokenExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

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
  // Ref instead of state — sendMessage always reads the live socket without
  // needing to be recreated when the socket changes.
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!userID || !accessToken || !packageSlug) return;

    if (isTokenExpired(accessToken)) {
      dispatch(logout());
      return;
    }

    const websocket = new WebSocket(
      `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(accessToken)}&packageSlug=${encodeURIComponent(packageSlug)}`,
    );
    wsRef.current = websocket;

    websocket.onopen = () => {};

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        dispatch(addError({ message: `Unknown WS message type received: "${(message as {type?:string})?.type ?? message}"`, timestamp: Date.now() }));
        return;
      }

      switch (message.type) {
        case TripEvents.DriverTripRequest:
          const trip = (message.data?.trip) ?? message.data;
          dispatch(setRequestedTrip(trip));
          dispatch(setTripStatus(message.type));
          // Subscribe to the trip topic so subsequent scoped events (chat, location) are delivered.
          if (trip?.id) {
            websocket.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${trip.id}` } }));
          }
          break;
        case TripEvents.DriverRegister:
          dispatch(setDriver(message.data));
          break;
        case TripEvents.ChatMessageReceived:
          dispatch(addChatMessage(message.data));
          break;
        case TripEvents.Cancelled:
          {
            const tripID = (message.data as { tripID?: string } | undefined)?.tripID;
            if (tripID) {
              websocket.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${tripID}` } }));
              websocket.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${tripID}:chat` } }));
            }
            dispatch(resetTrip());
            dispatch(setTripStatus(TripEvents.Cancelled));
          }
          break;
      }
    };

    websocket.onclose = (e: CloseEvent) => {
      wsRef.current = null;
      if (e.code === 1000 || e.code === 1001) return; // clean close, no error
      if (e.code === 1006) {
        // Abnormal closure — server rejected the upgrade (e.g. 401) or network dropped.
        if (isTokenExpired(accessToken)) {
          dispatch(logout());
        } else {
          dispatch(addError({ message: 'Lost connection to server. Please refresh.', timestamp: Date.now() }));
        }
      } else {
        dispatch(addError({ message: e.reason || `Connection closed unexpectedly (code ${e.code}).`, timestamp: Date.now() }));
      }
    };

    websocket.onerror = () => { /* onclose fires next with the close code — handled there */ };

    return () => {
      websocket.close();
      wsRef.current = null;
    };
  }, [userID, accessToken, packageSlug, dispatch]);

  // stable across renders — reads from ref so it always uses the live socket
  const sendMessage = useCallback((message: ClientWsMessage) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
    // silently drop if not yet open — watchPosition will retry on next tick
  }, []);

  return { sendMessage };
}
