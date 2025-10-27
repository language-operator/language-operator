# frozen_string_literal: true

class HealthController < ApplicationController
  skip_before_action :verify_authenticity_token

  def index
    render json: { status: 'ok', timestamp: Time.current }
  end
end
