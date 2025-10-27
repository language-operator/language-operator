require_relative "boot"

require "rails"
# Pick the frameworks you want:
require "active_model/railtie"
require "active_job/railtie"
require "active_record/railtie"
# require "active_storage/engine"
require "action_controller/railtie"
# require "action_mailer/railtie"
# require "action_mailbox/engine"
# require "action_text/engine"
require "action_view/railtie"
require "action_cable/engine"
# require "rails/test_unit/railtie"

# Require the gems listed in Gemfile, including any gems
# you've limited to :test, :development, or :production.
Bundler.require(*Rails.groups)

module Web
  class Application < Rails::Application
    # Initialize configuration defaults for originally generated Rails version.
    config.load_defaults 7.2

    # Please, add to the `ignore` list any other `lib` subdirectories that do
    # not contain `.rb` files, or that should not be reloaded or eager loaded.
    # Common ones are `templates`, `generators`, or `middleware`, for example.
    config.autoload_lib(ignore: %w[assets tasks])

    # Add based_client library path
    $LOAD_PATH.unshift('/app/lib')

    # Session store with Redis
    config.session_store :redis_session_store,
                         key: '_based_web_session',
                         redis: {
                           url: ENV.fetch('REDIS_URL', 'redis://redis:6379/0'),
                           expire_after: 24.hours
                         }

    # CORS configuration
    config.middleware.insert_before 0, Rack::Cors do
      allow do
        origins '*'
        resource '*',
                 headers: :any,
                 methods: %i[get post put patch delete options head],
                 credentials: false
      end
    end

    # Time zone
    config.time_zone = 'UTC'

    # Don't generate system test files.
    config.generators.system_tests = nil
  end
end
