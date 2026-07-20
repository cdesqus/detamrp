FROM node:24.11.1-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG API_URL=http://backend:8091
ENV API_URL=$API_URL
RUN npm run build

FROM node:24.11.1-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app ./
EXPOSE 3000
CMD ["npm", "start"]
