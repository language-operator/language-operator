# Web search and scraping tools for MCP

tool "web_search" do
  description "Search the web using DuckDuckGo and return results"

  parameter "query" do
    type :string
    required true
    description "The search query"
  end

  parameter "max_results" do
    type :number
    required false
    description "Maximum number of results to return (default: 5)"
    default 5
  end

  execute do |params|
    query = params["query"]
    max_results = (params["max_results"] || 5).to_i

    # URL encode the query
    encoded_query = URI.encode_www_form_component(query)

    # Use DuckDuckGo HTML interface
    url = "https://html.duckduckgo.com/html/?q=#{encoded_query}"

    # Fetch results using SDK HTTP client
    response = LanguageOperator::Dsl::HTTP.get(url, headers: { 'User-Agent' => 'Mozilla/5.0' }, follow_redirects: true)

    unless response[:success]
      next "Error: Failed to fetch search results - #{response[:error] || response[:status]}"
    end

    html = response[:body]

    # Parse results (simple text extraction)
    results = []

    # Extract result blocks using simple pattern matching
    html.scan(/<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>([^<]+)<\/a>/i).each_with_index do |(href, title), index|
      break if index >= max_results

      # Clean up the URL (DuckDuckGo redirects)
      clean_url = href.gsub(/^\/\/duckduckgo\.com\/l\/\?.*uddg=/, '')
      clean_url = URI.decode_www_form_component(clean_url) if clean_url.include?('%')

      results << "#{index + 1}. #{title.strip}\n   URL: #{clean_url}"
    end

    if results.empty?
      "No results found for: #{query}"
    else
      "Search results for: #{query}\n\n" + results.join("\n\n")
    end
  end
end

tool "web_fetch" do
  description "Fetch and extract text content from a URL"

  parameter "url" do
    type :string
    required true
    description "The URL to fetch"
  end

  parameter "html" do
    type :boolean
    required false
    description "Return raw HTML instead of text (default: false)"
    default false
  end

  execute do |params|
    url = params["url"]
    return_html = params["html"] || false

    # Validate URL
    unless url =~ /^https?:\/\//
      next "Error: Invalid URL. Must start with http:// or https://"
    end

    # Fetch the URL using SDK HTTP client
    response = LanguageOperator::Dsl::HTTP.get(url, headers: { 'User-Agent' => 'Mozilla/5.0' }, follow_redirects: true)

    unless response[:success]
      next "Error: Failed to fetch URL: #{url} - #{response[:error] || response[:status]}"
    end

    content = response[:body]

    if return_html
      content
    else
      # Strip HTML tags for text-only output
      text = content.gsub(/<script[^>]*>.*?<\/script>/im, '')
                   .gsub(/<style[^>]*>.*?<\/style>/im, '')
                   .gsub(/<[^>]+>/, ' ')
                   .gsub(/\s+/, ' ')
                   .strip

      if text.empty?
        "No text content found at: #{url}"
      else
        "Content from #{url}:\n\n#{text[0..2000]}#{text.length > 2000 ? '...' : ''}"
      end
    end
  end
end

tool "web_headers" do
  description "Fetch HTTP headers from a URL"

  parameter "url" do
    type :string
    required true
    description "The URL to check"
  end

  execute do |params|
    url = params["url"]

    # Validate URL
    unless url =~ /^https?:\/\//
      next "Error: Invalid URL. Must start with http:// or https://"
    end

    # Fetch headers using SDK HTTP client
    response = LanguageOperator::Dsl::HTTP.head(url)

    unless response[:success]
      next "Error: Failed to fetch headers from: #{url} - #{response[:error] || response[:status]}"
    end

    headers_text = response[:headers].map { |k, v| "#{k}: #{Array(v).join(', ')}" }.join("\n")
    "Headers for #{url}:\n\n#{headers_text}"
  end
end

tool "web_status" do
  description "Check the HTTP status code of a URL"

  parameter "url" do
    type :string
    required true
    description "The URL to check"
  end

  execute do |params|
    url = params["url"]

    # Validate URL
    unless url =~ /^https?:\/\//
      next "Error: Invalid URL. Must start with http:// or https://"
    end

    # Get status code using SDK HTTP client (don't follow redirects to get actual status)
    response = LanguageOperator::Dsl::HTTP.get(url, follow_redirects: false)

    status = response[:status] || 0

    status_text = case status
    when 200 then "OK"
    when 301 then "Moved Permanently"
    when 302 then "Found (Redirect)"
    when 304 then "Not Modified"
    when 400 then "Bad Request"
    when 401 then "Unauthorized"
    when 403 then "Forbidden"
    when 404 then "Not Found"
    when 500 then "Internal Server Error"
    when 502 then "Bad Gateway"
    when 503 then "Service Unavailable"
    else "Unknown"
    end

    "Status for #{url}: #{status} #{status_text}"
  end
end
