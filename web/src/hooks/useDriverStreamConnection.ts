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
  setError,
} from '../store/slices/driverSlice';

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

    const websocket = new WebSocket(
      `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(accessToken)}&packageSlug=${encodeURIComponent(packageSlug)}`,
    );
    wsRef.current = websocket;

    websocket.onopen = () => {};

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        dispatch(setError(`Unknown message type "${message}", allowed types are: ${Object.values(TripEvents).join(', ')}`));
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
      }
    };

    websocket.onclose = () => {
      wsRef.current = null;
    };

    websocket.onerror = () => {
      dispatch(setError('WebSocket error occurred'));
    };

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
