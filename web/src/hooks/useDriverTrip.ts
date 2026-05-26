import { useCallback, useEffect, useMemo, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import {
  setTripStatus,
  resetTrip,
} from '../store/slices/driverSlice';
import { addError } from '../store/slices/uiSlice';
import { CarPackageSlug, Coordinate } from '../types';
import { TripEvents } from '../contracts';
import { useDriverStreamConnection } from './useDriverStreamConnection';
import * as Geohash from 'ngeohash';

const START_LOCATION: Coordinate = {
  latitude: 0,
  longitude: 0,
};

export function useDriverTrip(packageSlug: CarPackageSlug) {
  const dispatch = useAppDispatch();
  const { driver, tripStatus, requestedTrip, error } = useAppSelector((s) => s.driver);
  const userID = useAppSelector((s) => s.auth.user?.id) ?? '';
  const accessToken = useAppSelector((s) => s.auth.accessToken) ?? '';
  const [driverLocation, setDriverLocation] = useState<Coordinate>(START_LOCATION);
  const [locationReady, setLocationReady] = useState(false);

  const { sendMessage } = useDriverStreamConnection({
    userID,
    accessToken,
    packageSlug,
  });

  // Switch to watchPosition so location updates continuously.
  // Each new position: update the UI marker AND push to the backend over WS.
  useEffect(() => {
    if (!navigator.geolocation) {
      return;
    }
    const watchId = navigator.geolocation.watchPosition(
      (pos) => {
        const coord: Coordinate = { latitude: pos.coords.latitude, longitude: pos.coords.longitude };
        setDriverLocation(coord);
        setLocationReady(true);
        const gh = Geohash.encode(coord.latitude, coord.longitude, 7);
        sendMessage({
          type: TripEvents.DriverLocation,
          data: { location: coord, geohash: gh },
        });
      },
      () => {
        dispatch(addError({ message: 'Unable to retrieve your location. Please allow location access.', timestamp: Date.now() }));
      },
    );
    return () => navigator.geolocation.clearWatch(watchId);
  }, [sendMessage, dispatch]);

  const handleAcceptTrip = useCallback(() => {
    if (!requestedTrip || !requestedTrip.id) {
      alert("No trip ID found");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
      },
    });

    dispatch(setTripStatus(TripEvents.DriverTripAccept));
  }, [requestedTrip, sendMessage, dispatch]);

  const handleDeclineTrip = useCallback(() => {
    if (!requestedTrip || !requestedTrip.id) {
      alert("No trip ID found");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
      },
    });

    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${requestedTrip.id}` },
    });
    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${requestedTrip.id}:chat` },
    });

    dispatch(setTripStatus(TripEvents.DriverTripDecline));
    dispatch(resetTrip());
  }, [requestedTrip, sendMessage, dispatch]);

  const handleCancelTrip = useCallback(() => {
    if (!requestedTrip || !requestedTrip.id) {
      return;
    }

    // Reuse the decline command as driver-side cancellation for the current trip context.
    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
      },
    });

    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${requestedTrip.id}` },
    });
    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${requestedTrip.id}:chat` },
    });

    dispatch(resetTrip());
  }, [requestedTrip, sendMessage, dispatch]);

  const driverGeohash = useMemo(
    () => Geohash.encode(driverLocation.latitude, driverLocation.longitude, 7),
    [driverLocation.latitude, driverLocation.longitude],
  );

  const parsedRoute = useMemo(
    () =>
      requestedTrip?.route?.geometry[0]?.coordinates
        .map((coord) => [coord?.longitude, coord?.latitude] as [number, number]),
    [requestedTrip],
  );

  const routeDestination = useMemo(
    () => requestedTrip?.route?.geometry[0]?.coordinates[requestedTrip?.route?.geometry[0]?.coordinates?.length - 1],
    [requestedTrip],
  );

  const routeStart = useMemo(
    () => requestedTrip?.route?.geometry[0]?.coordinates[0],
    [requestedTrip],
  );

  return {
    userID,
    sendMessage,
    driver,
    tripStatus,
    requestedTrip,
    error,
    driverLocation,
    locationReady,
    driverGeohash,
    parsedRoute,
    routeDestination,
    routeStart,
    handleAcceptTrip,
    handleDeclineTrip,
    handleCancelTrip,
  };
}
