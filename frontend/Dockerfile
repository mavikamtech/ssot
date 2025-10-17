# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

# Accept build arguments
ARG NEXT_PUBLIC_GQL_ENDPOINT
ARG NEXT_PUBLIC_AZURE_AD_TENANT_ID
ARG NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI

# Set environment variables for the build
ENV NEXT_PUBLIC_GQL_ENDPOINT=$NEXT_PUBLIC_GQL_ENDPOINT
ENV NEXT_PUBLIC_AZURE_AD_TENANT_ID=$NEXT_PUBLIC_AZURE_AD_TENANT_ID
ENV NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI=$NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM node:22-alpine AS runner

WORKDIR /app

# Install curl for health checks
RUN apk add --no-cache curl

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
ENV NODE_ENV=production
ENV NEXT_PUBLIC_AZURE_AD_TENANT_ID="0beed8a0-2f9c-4bff-a92a-1e445f1c15bd"
ENV NEXT_PUBLIC_POST_LOGOUT_REDIRECT_URI="https://app-prod.mavik-ssot.com/"

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD curl -f http://localhost:3000/health || exit 1

CMD ["node", "server.js"]