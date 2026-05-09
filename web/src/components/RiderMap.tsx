'use client';

import Image from 'next/image';
import { useRiderStreamConnection } from '../hooks/useRiderStreamConnection';
import { MapContainer, Marker, Popup, Rectangle, TileLayer } from 'react-leaflet'
import L from 'leaflet';
import { getGeohashBounds } from '../utils/geohash';
import { useRef } from 'react';
import { MapClickHandler } from './MapClickHandler';
import { RoutingControl } from "./RoutingControl";
import { RiderTripOverview } from './RiderTripOverview';
import { useAppSelector } from '../store/store';
import { useRiderTrip } from '../hooks/useRiderTrip';
import { UserInfo } from './UserInfo';

const userMarker = new L.Icon({
    iconUrl: "https://upload.wikimedia.org/wikipedia/commons/thumb/e/ed/Map_pin_icon.svg/176px-Map_pin_icon.svg.png",
    iconSize: [40, 40],
    iconAnchor: [20, 40],
});

const driverMarker = new L.Icon({
    iconUrl: "https://www.svgrepo.com/show/25407/car.svg",
    iconSize: [30, 30],
    iconAnchor: [15, 30],
});

interface RiderMapProps {
    onRouteSelected?: (distance: number) => void;
}

export default function RiderMap({ onRouteSelected }: RiderMapProps) {
    const mapRef = useRef<L.Map>(null);
    const userID = useAppSelector((s) => s.auth.user?.id) ?? '';
    const accessToken = useAppSelector((s) => s.auth.accessToken) ?? '';

    const location = {
        latitude: 12.920422,
        longitude: 77.611008,
    };

    useRiderStreamConnection(location, userID, accessToken);

    const { drivers, error, tripStatus, assignedDriver, paymentSession } = useAppSelector((s) => s.rider);
    const { trip, destination, handleMapClick, handleStartTrip, handleCancelTrip } = useRiderTrip(userID);

    const onMapClick = async (e: L.LeafletMouseEvent) => {
        await handleMapClick(e.latlng, location, onRouteSelected);
    };

    if (error) {
        return <div>Error: {error}</div>
    }

    return (
        <div className="relative flex flex-col md:flex-row h-screen">
            <UserInfo />
            <div className={`${destination ? 'flex-[0.7]' : 'flex-1'}`}>
                <MapContainer
                    center={[location.latitude, location.longitude]}
                    zoom={13}
                    style={{ height: '100%', width: '100%' }}
                    ref={mapRef}
                >
                    <TileLayer
                        url="https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png"
                        attribution="&copy; <a href='https://www.openstreetmap.org/copyright'>OpenStreetMap</a> contributors &copy; <a href='https://carto.com/'>CARTO</a>"
                    />
                    <Marker position={[location.latitude, location.longitude]} icon={userMarker} />

                    {drivers?.map((driver) => (
                        <Rectangle
                            key={`grid-${driver?.geohash}`}
                            bounds={getGeohashBounds(driver?.geohash) as L.LatLngBoundsExpression}
                            pathOptions={{
                                color: '#3388ff',
                                weight: 1,
                                fillOpacity: 0.1
                            }}
                        >
                            <Popup>Geohash: {driver?.geohash}</Popup>
                        </Rectangle>
                    ))}

                    {drivers?.map((driver) => (
                        <Marker
                            key={driver?.id}
                            position={[driver?.location?.latitude, driver?.location?.longitude]}
                            icon={driverMarker}
                        >
                            <Popup>
                                Driver ID: {driver?.id}
                                <br />
                                Geohash: {driver?.geohash}
                                <br />
                                Name: {driver?.name}
                                <br />
                                Car Plate: {driver?.carPlate}
                                <br />
                                <Image
                                    src={driver?.profilePicture}
                                    alt={`${driver?.name}'s profile picture`}
                                    width={100}
                                    height={100}
                                />
                            </Popup>
                        </Marker>
                    ))}
                    {destination && (
                        <Marker position={destination} icon={userMarker}>
                            <Popup>Destination</Popup>
                        </Marker>
                    )}

                    {trip && (
                        <RoutingControl route={trip.route} />
                    )}
                    <MapClickHandler onClick={onMapClick} />
                </MapContainer>
            </div>

            <div className="flex-[0.4]">
                <RiderTripOverview
                    trip={trip}
                    assignedDriver={assignedDriver}
                    status={tripStatus}
                    paymentSession={paymentSession}
                    onPackageSelect={handleStartTrip}
                    onCancel={handleCancelTrip}
                />
            </div>
        </div>
    )
}