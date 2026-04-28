# Multi-stage build untuk frontend.
# Stage 1: build static dist via Vite
# Stage 2: nginx serve dist + proxy /api ke backend

FROM node:20-alpine AS builder

WORKDIR /src

# Cache deps layer
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ---- Runtime stage ----
FROM nginx:1.27-alpine

COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
