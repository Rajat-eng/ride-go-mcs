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

  const getTripIdentifiers = useCallback(() => ({
    tripID: requestedTrip?.id ?? '',
    riderID: requestedTrip?.userID ?? '',
  }), [requestedTrip]);

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

  // Keep publishing the latest known location periodically so the driver
  // re-registers as available even after WS reconnects while stationary.
  useEffect(() => {
    if (!locationReady) {
      return;
    }

    const pushLocation = () => {
      const gh = Geohash.encode(driverLocation.latitude, driverLocation.longitude, 7);
      sendMessage({
        type: TripEvents.DriverLocation,
        data: { location: driverLocation, geohash: gh },
      });
    };

    // Fire once immediately, then keep alive every few seconds.
    pushLocation();
    const intervalID = window.setInterval(pushLocation, 5000);

    return () => window.clearInterval(intervalID);
  }, [locationReady, driverLocation, sendMessage]);

  const handleAcceptTrip = useCallback(() => {
    const { tripID, riderID } = getTripIdentifiers();
    if (!tripID || !riderID) {
      alert("No trip ID found");
      return;
    }

    const wasSent = sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID,
        riderID,
      },
    });

    if (!wasSent) {
      dispatch(addError({ message: 'Connection is not ready. Please wait and try again.', timestamp: Date.now() }));
      return;
    }

    dispatch(setTripStatus(TripEvents.DriverTripAccept));
  }, [dispatch, getTripIdentifiers, sendMessage]);

  const handleDeclineTrip = useCallback(() => {
    const { tripID, riderID } = getTripIdentifiers();
    if (!tripID || !riderID) {
      alert("No trip ID found");
      return;
    }

    const declineSent = sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID,
        riderID,
      },
    });
    if (!declineSent) {
      dispatch(addError({ message: 'Connection is not ready. Please wait and try again.', timestamp: Date.now() }));
      return;
    }

    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${tripID}` },
    });
    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${tripID}:chat` },
    });

    dispatch(setTripStatus(TripEvents.DriverTripDecline));
    dispatch(resetTrip());
  }, [dispatch, getTripIdentifiers, sendMessage]);

  const handleCancelTrip = useCallback(() => {
    const { tripID, riderID } = getTripIdentifiers();
    if (!tripID || !riderID) {
      return;
    }

    // Reuse the decline command as driver-side cancellation for the current trip context.
    const cancelSent = sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID,
        riderID,
      },
    });
    if (!cancelSent) {
      dispatch(addError({ message: 'Connection is not ready. Please wait and try again.', timestamp: Date.now() }));
      return;
    }

    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${tripID}` },
    });
    sendMessage({
      type: TripEvents.WsTopicUnsubscribe,
      data: { topic: `trip:${tripID}:chat` },
    });

    dispatch(resetTrip());
  }, [dispatch, getTripIdentifiers, sendMessage]);

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
