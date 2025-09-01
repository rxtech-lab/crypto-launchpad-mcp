import { InstallationStep, InstallationType } from "@/types/quick-start";

export const INSTALLATION_STEPS: Record<InstallationType, InstallationStep[]> = {
  local: [
    {
      step: 1,
      title: "Download and Install",
      description:
        "Download the latest release for macOS and run the installer.",
      code: "# The binary will be installed to\n/usr/local/bin/launchpad-mcp",
    },
    {
      step: 2,
      title: "Configure Your Client",
      description:
        "Add the MCP server to your preferred AI client configuration.",
      hasIDETabs: true,
    },
    {
      step: 3,
      title: "Start Using",
      description:
        "Restart Claude Desktop and start deploying tokens with natural language.",
      benefits: [
        "Deploy ERC-20 tokens on Ethereum",
        "Create Uniswap liquidity pools",
        "Manage liquidity positions",
        "Execute token swaps",
      ],
    },
  ],
  remote: [
    {
      step: 1,
      title: "Choose Remote MCP Server",
      description:
        "Use our hosted MCP server instead of installing locally. Perfect for cloud-based workflows.",
      code: `{
  "launchpad-mcp": {
    "url": "https://launchpad.mcprouter.app/mcp",
    "type": "http"
  }
}`,
    },
    {
      step: 2,
      title: "Configure Your IDE/Client",
      description:
        "Add the remote MCP server to your preferred IDE or AI client.",
      hasIDETabs: true,
    },
    {
      step: 3,
      title: "Start Using Remote MCP",
      description:
        "Restart your IDE/client and start deploying tokens through the cloud.",
      benefits: [
        "No local installation required",
        "Always up-to-date server version",
        "Cross-platform compatibility",
        "Scalable cloud infrastructure",
      ],
    },
  ],
};