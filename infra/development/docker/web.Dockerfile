FROM node:20-alpine

WORKDIR /app

ARG NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=""
ENV NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=${NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY}

COPY web/package*.json ./

RUN npm install

COPY web ./

RUN npm run build

EXPOSE 3000

CMD ["npm", "start"]