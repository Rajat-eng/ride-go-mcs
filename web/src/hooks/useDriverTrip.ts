import { useCallback, useMemo, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../store/store';
import {
  setTripStatus,
  resetTrip,
} from '../store/slices/driverSlice';
import { CarPackageSlug, Coordinate } from '../types';
import { TripEvents } from '../contracts';
import { useDriverStreamConnection } from './useDriverStreamConnection';
import * as Geohash from 'ngeohash';

const START_LOCATION: Coordinate = {
  latitude: 37.7749,
  longitude: -122.4194,
};

export function useDriverTrip(packageSlug: CarPackageSlug) {
  const dispatch = useAppDispatch();
  const { driver, tripStatus, requestedTrip, error } = useAppSelector((s) => s.driver);
  const userID = useAppSelector((s) => s.auth.user?.id) ?? '';
  const accessToken = useAppSelector((s) => s.auth.accessToken) ?? '';
  const [driverLocation, setDriverLocation] = useState<Coordinate>(START_LOCATION);

  const driverGeohash = useMemo(
    () => Geohash.encode(driverLocation.latitude, driverLocation.longitude, 7),
    [driverLocation.latitude, driverLocation.longitude],
  );

  const { sendMessage } = useDriverStreamConnection({
    location: driverLocation,
    geohash: driverGeohash,
    userID,
    accessToken,
    packageSlug,
  });

  const handleMapClick = useCallback((latlng: { lat: number; lng: number }) => {
    setDriverLocation({ latitude: latlng.lat, longitude: latlng.lng });
  }, []);

  const handleAcceptTrip = useCallback(() => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripAccept,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      },
    });

    dispatch(setTripStatus(TripEvents.DriverTripAccept));
  }, [requestedTrip, driver, sendMessage, dispatch]);

  const handleDeclineTrip = useCallback(() => {
    if (!requestedTrip || !requestedTrip.id || !driver) {
      alert("No trip ID found or driver is not set");
      return;
    }

    sendMessage({
      type: TripEvents.DriverTripDecline,
      data: {
        tripID: requestedTrip.id,
        riderID: requestedTrip.userID,
        driver,
      },
    });

    dispatch(setTripStatus(TripEvents.DriverTripDecline));
    dispatch(resetTrip());
  }, [requestedTrip, driver, sendMessage, dispatch]);

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
    driver,
    tripStatus,
    requestedTrip,
    error,
    driverLocation,
    driverGeohash,
    parsedRoute,
    routeDestination,
    routeStart,
    handleMapClick,
    handleAcceptTrip,
    handleDeclineTrip,
  };
}
