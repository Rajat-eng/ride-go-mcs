import { useEffect, useState } from 'react';
import { WEBSOCKET_URL } from "../constants";
import { CarPackageSlug } from '../types';
import { ServerWsMessage, TripEvents, isValidWsMessage, isValidTripEvent, ClientWsMessage, BackendEndpoints } from '../contracts';
import { useAppDispatch } from '../store/store';
import {
  setDriver,
  setRequestedTrip,
  setTripStatus,
  setError,
} from '../store/slices/driverSlice';

interface useDriverConnectionProps {
  location: {
    latitude: number;
    longitude: number;
  };
  geohash: string;
  userID: string;
  accessToken: string;
  packageSlug: CarPackageSlug;
}

export const useDriverStreamConnection = ({
  location,
  geohash,
  userID,
  accessToken,
  packageSlug
}: useDriverConnectionProps) => {
  const dispatch = useAppDispatch();
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    if (!userID || !accessToken) return;

    const websocket = new WebSocket(
      `${WEBSOCKET_URL}${BackendEndpoints.WS_DRIVERS}?token=${encodeURIComponent(accessToken)}&packageSlug=${encodeURIComponent(packageSlug)}`,
    );
    setWs(websocket);

    websocket.onopen = () => {
      if (location) {
        websocket.send(JSON.stringify({
          type: TripEvents.DriverLocation,
          data: { location, geohash },
        }));
      }
    };

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
          break;
        case TripEvents.DriverRegister:
          dispatch(setDriver(message.data));
          break;
      }

      if (isValidTripEvent(message.type)) {
        dispatch(setTripStatus(message.type));
      } else {
        dispatch(setError(`Unknown message type "${message.type}", allowed types are: ${Object.values(TripEvents).join(', ')}`));
      }
    };

    websocket.onclose = () => {};

    websocket.onerror = () => {
      dispatch(setError('WebSocket error occurred'));
    };

    return () => {
      if (websocket.readyState === WebSocket.OPEN) {
        websocket.close();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userID, accessToken, packageSlug, location, geohash, dispatch]);

  const sendMessage = (message: ClientWsMessage) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      dispatch(setError('WebSocket is not connected'));
    }
  };

  return { sendMessage };
}
