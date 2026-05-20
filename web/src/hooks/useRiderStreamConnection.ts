import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { Coordinate } from '../types';
import { TripEvents, ServerWsMessage, isValidWsMessage, BackendEndpoints } from '../contracts';
import { useAppDispatch } from '../store/store';
import {
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setAssignedDriverLocation,
  setError,
} from '../store/slices/riderSlice';

export function useRiderStreamConnection(location: Coordinate | null, userID: string, accessToken: string) {
  const dispatch = useAppDispatch();
  const wsRef = useRef<WebSocket | null>(null);

  // Open the WS once on auth — location is NOT a dependency here.
  useEffect(() => {
    if (!userID || !accessToken) return;

    const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?token=${encodeURIComponent(accessToken)}`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        dispatch(setError(`Unknown message type "${message}", allowed types are: ${Object.values(TripEvents).join(', ')}`));
        return;
      }

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
          break;
        case TripEvents.DriverAssigned:
          dispatch(setAssignedDriver(message.data.driver ?? null));
          dispatch(setTripStatus(message.type));
          break;
        case TripEvents.Created:
          dispatch(setTripStatus(message.type));
          break;
        case TripEvents.NoDriversFound:
          dispatch(setTripStatus(message.type));
          break;
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
    };

    ws.onerror = () => {
      dispatch(setError('WebSocket error occurred'));
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [userID, accessToken, dispatch]);

  // Send location over the existing WS as soon as it is available.
  // Runs whenever location changes without touching the WS lifecycle.
  useEffect(() => {
    if (!location) return;
    const ws = wsRef.current;
    if (!ws) return;

    const send = () => ws.send(JSON.stringify({
      type: TripEvents.DriverLocation,
      data: { location },
    }));

    if (ws.readyState === WebSocket.OPEN) {
      send();
    } else {
      ws.addEventListener('open', send, { once: true });
    }
  }, [location]);

  const sendMessage = useCallback((message: object) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }, []);

  return { sendMessage };
}
