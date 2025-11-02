require_relative 'server'

# Completely bypass Rack::Protection by not using Sinatra's run! method
run Langop::Server
