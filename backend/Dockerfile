# Stage 1: Build the frontend (assuming it's a static site or vite/react/etc)
FROM node:20-alpine AS frontend-build
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Backend (Node.js)
FROM node:20-alpine
WORKDIR /app

# Copy backend files
COPY backend/package*.json ./backend/
RUN cd backend && npm ci

# Copy frontend build output if your backend serves it
COPY --from=frontend-build /app/frontend/dist ./frontend/dist  # ← adjust 'dist' to your build folder name

# Copy rest of backend
COPY backend/ ./backend/

# Optional: copy any root config if needed
# COPY .env* ./

WORKDIR /app/backend
EXPOSE 3000  # change if your port is different

CMD ["npm", "start"]
