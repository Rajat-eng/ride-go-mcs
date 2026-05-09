import { useEffect } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { Coordinate } from '../types';
import { TripEvents, ServerWsMessage, isValidWsMessage, BackendEndpoints } from '../contracts';
import { useAppDispatch } from '../store/store';
import {
  setDrivers,
  setTripStatus,
  setPaymentSession,
  setAssignedDriver,
  setError,
} from '../store/slices/riderSlice';

export function useRiderStreamConnection(location: Coordinate, userID: string, accessToken: string) {
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (!userID || !accessToken) return;

    const ws = new WebSocket(`${WEBSOCKET_URL}${BackendEndpoints.WS_RIDERS}?token=${encodeURIComponent(accessToken)}`);

    ws.onopen = () => {
      if (location) {
        ws.send(JSON.stringify({
          type: TripEvents.DriverLocation,
          data: { location },
        }));
      }
    };

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

    ws.onclose = () => {};

    ws.onerror = () => {
      dispatch(setError('WebSocket error occurred'));
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
  }, [userID, accessToken, location, dispatch]);
}
