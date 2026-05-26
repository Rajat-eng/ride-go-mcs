import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { TripEvents, ServerWsMessage, isValidWsMessage, BackendEndpoints } from '../contracts';
import { useAppDispatch } from '../store/store';
import {
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setAssignedDriverLocation,
  addChatMessage,
  resetTrip,
} from '../store/slices/riderSlice';
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

export function useRiderStreamConnection(userID: string, accessToken: string) {
  const dispatch = useAppDispatch();
  const wsRef = useRef<WebSocket | null>(null);

  // Open the WS once on auth — location is NOT a dependency here.
  useEffect(() => {
    if (!userID || !accessToken) return;

    if (isTokenExpired(accessToken)) {
      dispatch(logout());
      return;
    }

    const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?token=${encodeURIComponent(accessToken)}`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerWsMessage;

      if (!message || !isValidWsMessage(message)) {
        dispatch(addError({ message: `Unknown WS message type received: "${(message as {type?:string})?.type ?? message}"`, timestamp: Date.now() }));
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
          // TripEventCreated data shape is { trip: Trip, pickupLat, pickupLng } — trip ID is nested.
          // message.data.trip.id is the canonical trip ID at runtime.
          {
            const tripID = message.data.trip?.id ?? (message.data as { id?: string }).id;
            if (tripID) {
              ws.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}` } }));
              ws.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}:chat` } }));
            }
          }
          break;
        case TripEvents.NoDriversFound:
          dispatch(setTripStatus(message.type));
          break;
        case TripEvents.ChatMessageReceived:
          dispatch(addChatMessage(message.data));
          break;
        case TripEvents.Cancelled:
          dispatch(resetTrip());
          dispatch(setTripStatus(TripEvents.Cancelled));
          // Unsubscribe from trip rooms so the in-memory room registry on the gateway
          // is cleaned up even if the cancel came from outside (e.g. admin).
          {
            const tripID = (message.data as { tripID?: string } | undefined)?.tripID;
            if (tripID) {
              ws.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${tripID}` } }));
              ws.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${tripID}:chat` } }));
            }
          }
          break;
      }

      if (message.type === TripEvents.ChatMessageReceived || message.type === TripEvents.DriverLocation || message.type === TripEvents.DriverEventLocation) {
        return;
      }
    };

    ws.onclose = (e: CloseEvent) => {
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

    ws.onerror = () => { /* onclose fires next with the close code — handled there */ };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [userID, accessToken, dispatch]);

  const sendMessage = useCallback((message: object) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    }
  }, []);

  return { sendMessage };
}
