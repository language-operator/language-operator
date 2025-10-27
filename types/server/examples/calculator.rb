# Example tool: Calculator
# This file demonstrates the MCP DSL for defining tools

tool "calculator" do
  description "Performs basic arithmetic operations on two numbers"

  parameter "operation" do
    type :string
    required true
    description "The arithmetic operation to perform"
    enum ["add", "subtract", "multiply", "divide"]
  end

  parameter "a" do
    type :number
    required true
    description "The first number"
  end

  parameter "b" do
    type :number
    required true
    description "The second number"
  end

  execute do |params|
    a = params["a"]
    b = params["b"]

    result = case params["operation"]
    when "add"
      a + b
    when "subtract"
      a - b
    when "multiply"
      a * b
    when "divide"
      if b == 0
        "Error: Division by zero"
      else
        a / b.to_f
      end
    else
      "Unknown operation"
    end

    "Result: #{result}"
  end
end

tool "echo" do
  description "Returns the input message"

  parameter "message" do
    type :string
    required true
    description "The message to echo back"
  end

  execute do |params|
    params["message"]
  end
end
