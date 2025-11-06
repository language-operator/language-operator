FROM git.theryans.io/langop/base:latest

# Install Ruby, build dependencies, Rails packages, and Node.js
RUN apk add --no-cache \
    ruby \
    ruby-dev \
    ruby-bundler \
    build-base \
    jq \
    curl \
    ca-certificates \
    nodejs \
    npm \
    tzdata \
    yaml-dev \
    sqlite-dev

# Set working directory
WORKDIR /app/web

# Copy Gemfile first for better Docker caching
COPY Gemfile Gemfile.lock* ./

# Configure bundle credentials and install
ARG REGISTRY_USERNAME
ARG REGISTRY_PASSWORD
RUN BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" \
    bundle lock --add-platform x86_64-linux && \
    BUNDLE_GIT__THERYANS__IO="${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}" \
    bundle install --no-cache

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
RUN chown -R langop:langop /app/web

# Switch to langop user
USER langop

# Expose ports (3000 for Rails, 3036 for Vite dev server)
EXPOSE 3000 3036

# Set environment
ENV RAILS_ENV=development \
    RAILS_LOG_TO_STDOUT=true \
    PORT=3000 \
    VITE_RUBY_AUTO_BUILD=true \
    WORKSPACE_PATH="/workspace"

# Start Rails server (Vite will auto-build assets in development)
CMD ["bundle", "exec", "rails", "server", "-b", "0.0.0.0"]
