# Build Stage
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy source and build production bundle
COPY . .
RUN npm run build

# Run Stage
FROM nginx:alpine

# Copy the compiled Vite assets to the Nginx html directory
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose standard web port
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
