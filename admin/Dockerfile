# syntax=docker/dockerfile:1

# --- stage 1: собрать статический бандл CRA ---
FROM node:20-alpine AS build
WORKDIR /app
ARG REACT_APP_PRODUCT_SITE_ORIGIN=https://shop.example.com
ENV REACT_APP_PRODUCT_SITE_ORIGIN=$REACT_APP_PRODUCT_SITE_ORIGIN
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# CRA treats warnings as errors when CI=true; keep production builds deterministic.
ENV CI=false
RUN npm run build

# --- stage 2: nginx раздаёт статику + проксирует /api на движок ---
FROM nginxinc/nginx-unprivileged:alpine3.23
COPY --from=build /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 8080
