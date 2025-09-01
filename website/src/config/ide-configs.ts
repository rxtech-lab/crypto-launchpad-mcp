import { Terminal, Code2, FileCode2, Blocks } from "lucide-react";
import { IDEConfig } from "@/types/quick-start";

export const IDE_CONFIGS: IDEConfig[] = [
  {
    id: "claude",
    name: "Claude",
    icon: Terminal,
    local: {
      description: "Add to your Claude Desktop configuration file:",
      code: `{
  "mcpServers": {
    "launchpad-mcp": {
      "command": "/usr/local/bin/launchpad-mcp",
      "args": []
    }
  }
}`,
      note: "Location: ~/Library/Application Support/Claude/claude_desktop_config.json",
    },
  },
  {
    id: "cursor",
    name: "Cursor",
    icon: FileCode2,
    local: {
      description: "Add to your Cursor settings:",
      code: `{
  "mcpServers": {
    "launchpad-mcp": {
      "command": "/usr/local/bin/launchpad-mcp",
      "args": [],
      "env": {}
    }
  }
}`,
      note: "Open Settings → MCP → Add Server",
    },
  },
  {
    id: "claude-code",
    name: "Claude Code",
    icon: Terminal,
    remote: {
      description: "Add the remote MCP server using Claude Code CLI:",
      command:
        "claude mcp add --transport http --scope local launchpad-mcp https://launchpad.mcprouter.app/mcp",
      note: "Requires Claude Code CLI and Pro/Team/Enterprise plan",
    },
  },
  {
    id: "claude-desktop",
    name: "Claude Desktop",
    icon: Terminal,
    remote: {
      description: "Add via Claude Desktop Settings (recommended):",
      code: `1. Open Claude Desktop
2. Go to Settings > Connectors
3. Add new connector with URL: https://launchpad.mcprouter.app/mcp`,
      alternativeDescription:
        "Alternative: Use mcp-remote proxy in claude_desktop_config.json",
      alternativeCode: `{
  "mcpServers": {
    "launchpad-mcp": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "https://launchpad.mcprouter.app/mcp"]
    }
  }
}`,
    },
  },
  {
    id: "cursor-remote",
    name: "Cursor",
    icon: FileCode2,
    remote: {
      description: "Add to your Cursor MCP settings:",
      code: `{
  "mcpServers": {
    "launchpad-mcp": {
      "url": "https://launchpad.mcprouter.app/mcp"
    }
  }
}`,
      note: 'Open Command Palette (Ctrl+Shift+P) → Search "Cursor Settings" → MCP',
    },
  },
  {
    id: "vscode",
    name: "VS Code",
    icon: Code2,
    remote: {
      description: "Create .vscode/mcp.json in your workspace:",
      code: `{
  "servers": {
    "launchpad-mcp": {
      "type": "http",
      "url": "https://launchpad.mcprouter.app/mcp"
    }
  }
}`,
      note: "Or use Command Palette: MCP: Add Server → HTTP Server",
    },
  },
  {
    id: "codex",
    name: "Codex CLI",
    icon: Terminal,
    remote: {
      description: "Add to ~/.codex/config.toml:",
      code: `[mcp_servers.launchpad-mcp]
type = "remote"
url = "https://launchpad.mcprouter.app/mcp"`,
      note: "Requires Codex CLI from OpenAI",
    },
  },
  {
    id: "gemini",
    name: "Gemini CLI",
    icon: Blocks,
    remote: {
      description: "Add to ~/.gemini/settings.json or .gemini/settings.json:",
      code: `{
  "mcpServers": {
    "launchpad-mcp": {
      "httpUrl": "https://launchpad.mcprouter.app/mcp"
    }
  }
}`,
      note: "Requires Google Gemini CLI",
    },
  },
];