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
  resetTrip,
} from '../store/slices/driverSlice';
import { addError } from '../store/slices/uiSlice';

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

  useEffect(() => {
    if (!userID || !accessToken || !packageSlug) return;

    dispatch(setOwnerUserID(userID));

    const websocket = new WebSocket(
      `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(accessToken)}&packageSlug=${encodeURIComponent(packageSlug)}`,
    );
    wsRef.current = websocket;

    websocket.onopen = () => {};

    websocket.onmessage = (event: MessageEvent<string>) => {
      const result = parseWsMessage(event.data);

      if (!result.ok) {
        dispatch(addError({ message: result.error, timestamp: Date.now() }));
        return;
      }

      const message = result.message;

      switch (message.type) {
        case TripEvents.DriverTripRequest: {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const trip = message.data as any;
          dispatch(setRequestedTrip(trip));
          dispatch(setTripStatus(message.type));
          if (trip?.id) {
            websocket.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${trip.id}` } }));
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
            websocket.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}` } }));
            websocket.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}:chat` } }));
          }
          dispatch(resetTrip());
          dispatch(setTripStatus(TripEvents.Cancelled));
          break;
        }
      }
    };

    websocket.onclose = (e: CloseEvent) => {
      wsRef.current = null;
      if (e.code === 1000 || e.code === 1001) return; // clean close
      dispatch(addError({ message: e.reason || `Connection closed unexpectedly (code ${e.code}).`, timestamp: Date.now() }));
    };

    websocket.onerror = () => { /* onclose fires next with the close code — handled there */ };

    return () => {
      websocket.close();
      wsRef.current = null;
    };
  }, [userID, accessToken, packageSlug, dispatch]);

  // stable across renders — reads from ref so it always uses the live socket
  const sendMessage = useCallback((message: ClientWsMessage): boolean => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return false;
    }
    const validation = validateClientMessage(message);
    if (!validation.ok) {
      dispatch(addError({ message: validation.error, timestamp: Date.now() }));
      return false;
    }
    ws.send(JSON.stringify(validation.payload));
    return true;
  }, [dispatch]);

  return { sendMessage };
}
