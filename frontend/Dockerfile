FROM node:22-alpine AS deps
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /src
COPY --from=deps /src/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN adduser -D -u 10001 app
COPY --from=builder /src/public ./public
COPY --from=builder /src/.next/standalone ./
COPY --from=builder /src/.next/static ./.next/static
USER app
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
