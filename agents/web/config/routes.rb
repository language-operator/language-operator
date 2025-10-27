Rails.application.routes.draw do
  # Health check endpoint
  get "up" => "rails/health#show", as: :rails_health_check
  get "health" => "health#index"

  # API routes
  namespace :api do
    resources :chats, only: %i[create show destroy] do
      member do
        get :history
        delete :history, action: :clear_history
      end
      resources :messages, only: [:create] do
        collection do
          get :create  # Allow GET for EventSource streaming
        end
      end
    end
    resources :tools, only: [:index]
    resources :servers, only: [:index]
  end

  # ActionCable endpoint
  mount ActionCable.server => '/cable'

  # Root path serves the web interface
  root 'application#index'
end
