FROM based/client:latest

# Switch back to root to install packages
USER root

# Install additional packages for Rails and Vite
RUN apk add --no-cache \
    nodejs \
    npm \
    tzdata \
    yaml-dev \
    sqlite-dev

# Set working directory
WORKDIR /app/web

# Copy Gemfile first for better Docker caching
COPY Gemfile ./
RUN bundle lock --add-platform x86_64-linux && bundle install --no-cache

# Copy package files and install npm dependencies
COPY package.json package-lock.json* ./
RUN npm install

# Copy Vite config before running vite install
COPY vite.config.js config/vite.json ./

# Copy entire Rails application
COPY . .

# Initialize Vite Rails (creates config/initializers/vite_ruby.rb if needed)
RUN bundle exec vite install || true

# Create public/vite directory for assets
RUN mkdir -p public/vite public/vite-dev public/vite-test

# Build Vite assets for development (pre-build to avoid runtime compilation)
RUN npm run build || echo "Vite build skipped - will run in development mode"

# Set ownership
RUN chown -R based:based /app/web

# Switch back to based user
USER based

# Expose ports (3000 for Rails, 3036 for Vite dev server)
EXPOSE 3000 3036

# Set environment
ENV RAILS_ENV=development \
    RAILS_LOG_TO_STDOUT=true \
    PORT=3000 \
    VITE_RUBY_AUTO_BUILD=true

# Start Rails server (Vite will auto-build assets in development)
CMD ["bundle", "exec", "rails", "server", "-b", "0.0.0.0"]
