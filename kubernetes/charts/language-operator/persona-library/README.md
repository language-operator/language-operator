# Language Operator Persona Library

This directory contains built-in personas that ship with the language-operator Helm chart. These personas provide pre-configured personalities and capabilities for common use cases.

## Available Personas

### 1. Financial Analyst (`financial-analyst`)
**Tone:** Professional
**Description:** Expert financial analyst specializing in market analysis, financial modeling, and investment research

**Capabilities:**
- Financial statement analysis and ratio calculations
- Market trend analysis and technical indicators
- DCF and comparable company valuation models
- Portfolio optimization and risk assessment
- Economic research and macroeconomic analysis
- Investment thesis development
- Financial forecasting and scenario modeling

**Best for:** Market analysis, investment research, financial modeling, portfolio management

---

### 2. DevOps Engineer (`devops-engineer`)
**Tone:** Technical
**Description:** Experienced DevOps engineer specializing in infrastructure automation, CI/CD, and cloud platforms

**Capabilities:**
- Kubernetes cluster management and deployment strategies
- CI/CD pipeline design with GitHub Actions, GitLab CI, Jenkins
- Infrastructure as Code with Terraform, CloudFormation, Pulumi
- Container orchestration and Docker best practices
- Cloud platform expertise (AWS, GCP, Azure)
- Monitoring and observability with Prometheus, Grafana, ELK
- Configuration management with Ansible, Chef, Puppet
- GitOps workflows and progressive delivery
- Security scanning and compliance automation

**Best for:** Infrastructure automation, CI/CD pipelines, cloud deployments, Kubernetes management

---

### 3. General Assistant (`general-assistant`)
**Tone:** Friendly
**Description:** Versatile AI assistant for general-purpose tasks, research, and problem-solving

**Capabilities:**
- General knowledge and research across diverse topics
- Creative problem-solving and brainstorming
- Writing assistance (emails, documents, summaries)
- Learning and education support
- Task planning and organization
- Basic calculations and data analysis
- Information synthesis and summarization

**Best for:** General research, writing assistance, learning support, task planning

---

### 4. Executive Assistant (`executive-assistant`)
**Tone:** Professional
**Description:** Professional executive assistant for scheduling, communication, and administrative support

**Capabilities:**
- Calendar management and meeting coordination
- Email drafting and professional correspondence
- Travel planning and itinerary creation
- Meeting preparation and agenda creation
- Document organization and filing
- Task prioritization and deadline tracking
- Research and information gathering
- Expense tracking and report preparation
- Stakeholder communication and relationship management

**Best for:** Executive support, scheduling, professional communications, administrative tasks

---

### 5. Customer Support (`customer-support`)
**Tone:** Friendly
**Description:** Empathetic customer support agent focused on resolving issues and ensuring customer satisfaction

**Capabilities:**
- Product knowledge and technical troubleshooting
- Issue diagnosis and resolution
- Account management and order assistance
- Refund and return processing guidance
- Product recommendations and feature explanations
- Complaint handling and de-escalation
- Documentation and knowledge base usage
- Ticket creation and case management
- Multi-channel support (email, chat, phone guidance)

**Best for:** Customer service, technical support, issue resolution, customer satisfaction

---

## Usage

### Using with aictl CLI

Create an agent with a persona from the library:

```bash
aictl agent create "Analyze the Q3 earnings report for AAPL" --persona financial-analyst
```

List available personas:

```bash
aictl persona list
```

View persona details:

```bash
aictl persona show financial-analyst
```

### Using with Kubernetes

The personas are deployed as ConfigMaps that can be referenced by LanguageAgent resources:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguageAgent
metadata:
  name: market-analyst
spec:
  instructions: "Analyze daily market trends and provide investment insights"
  persona: financial-analyst
  mode: scheduled
  schedule: "0 9 * * MON-FRI"
```

### Creating Custom Personas

You can create your own personas based on these templates:

```bash
# Create a custom persona based on financial-analyst
aictl persona create my-crypto-analyst --from financial-analyst
```

Or apply a custom persona directly:

```yaml
apiVersion: langop.io/v1alpha1
kind: LanguagePersona
metadata:
  name: my-custom-persona
spec:
  displayName: My Custom Persona
  description: Custom persona for specific use case
  tone: neutral
  systemPrompt: |
    Your custom system prompt here...
  capabilities:
    - Capability 1
    - Capability 2
```

## Configuration

The persona library can be enabled/disabled in the Helm chart values:

```yaml
personaLibrary:
  enabled: true
  personas:
    - financial-analyst
    - devops-engineer
    - general-assistant
    - executive-assistant
    - customer-support
```

To disable the persona library:

```yaml
personaLibrary:
  enabled: false
```

To install only specific personas:

```yaml
personaLibrary:
  enabled: true
  personas:
    - devops-engineer
    - general-assistant
```

## Implementation Details

Each persona ConfigMap contains:

- **displayName**: Human-readable name for the persona
- **description**: Brief description of the persona's purpose
- **tone**: Communication style (neutral, friendly, professional, technical, creative)
- **systemPrompt**: Detailed instructions for the LLM defining the persona's behavior
- **capabilities**: List of specific skills and knowledge areas
- **toolPreferences** (optional): Preferred tools for this persona
- **responseFormat** (optional): Preferred response structure and formatting

## Contributing

To add new personas to the library:

1. Create a new YAML file in this directory following the existing format
2. Add the persona to the `persona-library.yaml` template
3. Update this README with the new persona details
4. Test the persona with various use cases
5. Submit a pull request

## License

These personas are part of the language-operator project and are licensed under the same terms.
