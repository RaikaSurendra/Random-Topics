FROM nginx:1.25-alpine

# Install curl for health checks
RUN apk add --no-cache curl

# Copy configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Create cache directory
RUN mkdir -p /var/cache/nginx/api_cache && \
    chown -R nginx:nginx /var/cache/nginx

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
