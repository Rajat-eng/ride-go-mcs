"use client"

import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet'
import L from 'leaflet';
import { useRef } from "react";
import { CarPackageSlug } from "../types";
import { DriverTripOverview } from "./DriverTripOverview";
import { RoutingControl } from "./RoutingControl";
import { DriverCard } from "./DriverCard";
import { useDriverTrip } from "../hooks/useDriverTrip";
import { UserInfo } from './UserInfo';
import { TripEvents } from '../contracts';
import { useAppSelector } from '../store/store';

const driverMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
  iconSize: [30, 30],
  iconAnchor: [15, 30],
});

const startLocationMarker = new L.Icon({
  iconUrl: "https://www.svgrepo.com/show/535711/user.svg",
  iconSize: [30, 40],
  iconAnchor: [20, 40],
});

const destinationMarker = new L.Icon({
  iconUrl: "https://upload.wikimedia.org/wikipedia/commons/thumb/e/ed/Map_pin_icon.svg/176px-Map_pin_icon.svg.png",
  iconSize: [40, 40],
  iconAnchor: [20, 40],
});

export const DriverMap = ({ packageSlug }: { packageSlug: CarPackageSlug }) => {
  const mapRef = useRef<L.Map>(null);
  const chatMessages = useAppSelector((s) => s.driver.chatMessages);

  const {
    userID,
    sendMessage,
    driver,
    tripStatus,
    requestedTrip,
    driverLocation,
    locationReady,
    driverGeohash,
    parsedRoute,
    routeDestination,
    routeStart,
    handleAcceptTrip,
    handleDeclineTrip,
    handleCancelTrip,
  } = useDriverTrip(packageSlug);

  if (!locationReady) {
    return <div>Waiting for location...</div>
  }

  const handleSendChatMessage = (tripID: string, text: string) => {
    sendMessage({
      type: TripEvents.ChatMessageSend,
      data: {
        tripID,
        text,
        messageID: crypto.randomUUID(),
      },
    }, { reportNotReady: true, queueIfNotReady: true });
  };

  return (
    <div className="relative flex flex-col md:flex-row h-screen">
      <UserInfo />
      <div className="flex-1">
        <MapContainer
          center={[driverLocation.latitude, driverLocation.longitude]}
          zoom={13}
          style={{ height: '100%', width: '100%' }}
          ref={mapRef}
        >
          <TileLayer
            url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
            attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
          />

          <Marker
            key={userID}
            position={[driverLocation.latitude, driverLocation.longitude]}
            icon={driverMarker}
          >
            <Popup>
              Driver ID: {userID}
              <br />
              Geohash: {driverGeohash}
            </Popup>
          </Marker>

          {routeStart && (
            <Marker position={[routeStart.longitude, routeStart.latitude]} icon={startLocationMarker}>
              <Popup>Start Location</Popup>
            </Marker>
          )}

          {routeDestination && (
            <Marker position={[routeDestination.longitude, routeDestination.latitude]} icon={destinationMarker}>
              <Popup>Destination</Popup>
            </Marker>
          )}

          {parsedRoute && (
            <RoutingControl route={parsedRoute} />
          )}
        </MapContainer>
      </div>

      <div className="flex flex-col md:w-[400px] bg-white border-t md:border-t-0 md:border-l">
        <div className="p-4 border-b">
          <DriverCard driver={driver} packageSlug={packageSlug} />
        </div>
        <div className="flex-1 overflow-y-auto">
          <DriverTripOverview
            trip={requestedTrip}
            status={tripStatus}
            userID={userID}
            chatMessages={chatMessages}
            onSendChatMessage={handleSendChatMessage}
            onAcceptTrip={handleAcceptTrip}
            onDeclineTrip={handleDeclineTrip}
            onCancelTrip={handleCancelTrip}
          />
        </div>
      </div>
    </div>
  )
}
