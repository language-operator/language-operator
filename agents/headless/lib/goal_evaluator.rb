# frozen_string_literal: true

# Goal evaluation and tracking for headless agent
#
# Manages objectives, tracks progress, and determines when goals are complete.
class GoalEvaluator
  attr_reader :goals, :results, :iterations

  # Initialize evaluator with goals configuration
  #
  # @param goals [Hash] Goals configuration from YAML
  def initialize(goals)
    @goals = goals
    @objectives = goals['objectives'] || []
    @success_criteria = goals['success_criteria'] || []
    @max_iterations = goals['max_iterations'] || 50
    @timeout_minutes = goals['timeout_minutes'] || 10
    @start_time = Time.now
    @iterations = 0
    @results = []
    @completed_objectives = Set.new
  end

  # Check if all goals are complete
  #
  # @return [Boolean] True if all objectives met
  def all_goals_complete?
    # Check timeout
    return true if timeout_exceeded?

    # Check max iterations
    return true if @iterations >= @max_iterations

    # Check if all objectives completed
    @completed_objectives.size >= @objectives.size
  end

  # Get the next task to work on
  #
  # @return [String, nil] Next objective or nil if none remaining
  def next_task
    @objectives.each_with_index do |objective, index|
      return objective unless @completed_objectives.include?(index)
    end

    nil
  end

  # Record the result of a task
  #
  # @param task [String] The task that was executed
  # @param response [String] The LLM response
  # @return [void]
  def record_result(task, response)
    @iterations += 1

    result = {
      iteration: @iterations,
      task: task,
      response: response,
      timestamp: Time.now
    }

    @results << result

    # Mark objective as complete if it appears in the response
    # This is a simple heuristic - in production you'd want more sophisticated evaluation
    @objectives.each_with_index do |objective, index|
      if response_addresses_objective?(response, objective)
        @completed_objectives.add(index)
      end
    end
  end

  # Get progress summary
  #
  # @return [Hash] Progress information
  def progress
    {
      iterations: @iterations,
      max_iterations: @max_iterations,
      completed: @completed_objectives.size,
      total_objectives: @objectives.size,
      remaining_objectives: @objectives.size - @completed_objectives.size,
      elapsed_minutes: elapsed_minutes,
      timeout_minutes: @timeout_minutes
    }
  end

  private

  # Check if response addresses an objective
  #
  # @param response [String] LLM response
  # @param objective [String] Objective to check
  # @return [Boolean] True if objective appears addressed
  def response_addresses_objective?(response, objective)
    # Simple keyword matching - could be enhanced with semantic analysis
    keywords = objective.downcase.split(/\W+/).reject { |w| w.length < 3 }
    response_lower = response.downcase

    keywords.any? { |keyword| response_lower.include?(keyword) }
  end

  # Check if timeout has been exceeded
  #
  # @return [Boolean] True if timeout exceeded
  def timeout_exceeded?
    elapsed_minutes >= @timeout_minutes
  end

  # Calculate elapsed time in minutes
  #
  # @return [Float] Elapsed minutes
  def elapsed_minutes
    (Time.now - @start_time) / 60.0
  end
end
