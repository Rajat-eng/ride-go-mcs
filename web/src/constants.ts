const isProduction = process.env.NODE_ENV === 'production';

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? (isProduction ? 'https://api.rexy.co.in' : 'http://localhost:8081');
export const WEBSOCKET_URL = process.env.NEXT_PUBLIC_WEBSOCKET_URL ?? (isProduction ? 'wss://ws.rexy.co.in/ws' : 'ws://localhost:8082/ws');
