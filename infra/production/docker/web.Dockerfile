FROM node:20-alpine

WORKDIR /app

ARG NEXT_PUBLIC_API_URL="https://api.rexy.co.in"
ARG NEXT_PUBLIC_WEBSOCKET_URL="wss://ws.rexy.co.in/ws"
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_WEBSOCKET_URL=${NEXT_PUBLIC_WEBSOCKET_URL}

COPY web/package*.json ./
RUN npm ci

COPY web ./
RUN npm run build

EXPOSE 3000
CMD ["npm", "start"]
