import { useCallback, useEffect, useRef } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { TripEvents, BackendEndpoints } from '../contracts';
import { parseWsMessage } from '../lib/ws/parseWsMessage';
import { useAppDispatch } from '../store/store';
import {
  setOwnerUserID,
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setAssignedDriverLocation,
  addChatMessage,
  resetTrip,
} from '../store/slices/riderSlice';
import { addError } from '../store/slices/uiSlice';

export function useRiderStreamConnection(userID: string, accessToken: string) {
  const dispatch = useAppDispatch();
  const wsRef = useRef<WebSocket | null>(null);

  const joinTripRooms = (ws: WebSocket, tripID?: string) => {
    if (!tripID) return;
    ws.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}` } }));
    ws.send(JSON.stringify({ type: TripEvents.WsTopicSubscribe, data: { topic: `trip:${tripID}:chat` } }));
  };

  // Open the WS once on auth — location is NOT a dependency here.
  useEffect(() => {
    if (!userID || !accessToken) return;

    dispatch(setOwnerUserID(userID));

    const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?token=${encodeURIComponent(accessToken)}`);
    wsRef.current = ws;

    ws.onmessage = (event: MessageEvent<string>) => {
      const result = parseWsMessage(event.data);

      if (!result.ok) {
        dispatch(addError({ message: result.error, timestamp: Date.now() }));
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
          dispatch(setAssignedDriver(trip.driver ? { id: trip.driver.id ?? '', name: trip.driver.name ?? '', location: { latitude: 0, longitude: 0 }, geohash: '', profilePicture: '', carPlate: '' } : null));
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
            ws.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}` } }));
            ws.send(JSON.stringify({ type: TripEvents.WsTopicUnsubscribe, data: { topic: `trip:${cancelledTripID}:chat` } }));
          }
          break;
        }
      }
    };

    ws.onclose = (e: CloseEvent) => {
      wsRef.current = null;
      if (e.code === 1000 || e.code === 1001) return; // clean close
      dispatch(addError({ message: e.reason || `Connection closed unexpectedly (code ${e.code}).`, timestamp: Date.now() }));
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
