import { readFileSync } from "fs";
import { join } from "path";

interface HookInput {
  session_id: string;
  tool_name: string;
  tool_input: { command?: string };
}

interface Operation {
  description: string;
  pattern: string;
}

interface ApprovalsConfig {
  version: string;
  description: string;
  operations: Record<string, Operation>;
}

function main() {
  try {
    const input = readFileSync(0, "utf-8");
    const data: HookInput = JSON.parse(input);

    if (data.tool_name !== "Bash") return;

    const command = data.tool_input?.command ?? "";
    if (!command.trim()) return;

    const projectDir =
      process.env.CLAUDE_PROJECT_DIR ?? join(process.cwd(), "../..");
    const configPath = join(
      projectDir,
      ".claude",
      "hooks",
      "bash-approvals.json"
    );

    let config: ApprovalsConfig;
    try {
      config = JSON.parse(readFileSync(configPath, "utf-8"));
    } catch {
      return; // No config — fail silently, let normal flow continue
    }

    for (const [name, op] of Object.entries(config.operations)) {
      const regex = new RegExp(op.pattern);
      if (regex.test(command.trimStart())) {
        process.stderr.write(`[bash-approval] approved: ${name}\n`);
        process.stdout.write(JSON.stringify({ decision: "approve" }));
        return;
      }
    }
    // No match — no output, normal permission flow
  } catch {
    // Silent fail — never interrupt Claude if the hook errors
  }
}

main();
