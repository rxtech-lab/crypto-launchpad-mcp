import { getLatestRelease } from "@/lib/github";
import { HeroSection } from "@/components/hero-section";
import { DownloadSection } from "@/components/download-button";
import { FeaturesGrid } from "@/components/features-grid";
import { QuickStartSection } from "@/components/quick-start-section";
import { CTASection } from "@/components/cta-section";

export default async function Home() {
  const releaseInfo = await getLatestRelease();

  const features = [
    {
      iconName: "Rocket",
      title: "AI-Powered Deployment",
      description:
        "Deploy tokens and manage liquidity pools using natural language commands through Claude AI.",
    },
    {
      iconName: "Shield",
      title: "Secure Wallet Integration",
      description:
        "EIP-6963 wallet discovery ensures secure client-side transaction signing with your favorite wallet.",
    },
    {
      iconName: "Zap",
      title: "Multi-Chain Support",
      description:
        "Seamlessly deploy on Ethereum and Solana with more chains coming soon.",
    },
    {
      iconName: "Coins",
      title: "Uniswap Integration",
      description:
        "Complete DEX functionality including pool creation, liquidity management, and token swaps.",
    },
    {
      iconName: "Code2",
      title: "Smart Contract Templates",
      description:
        "Pre-built OpenZeppelin templates and custom contracts for quick deployment.",
    },
    {
      iconName: "GitBranch",
      title: "MCP Protocol",
      description:
        "Built on Model Context Protocol for seamless AI integration and tool orchestration.",
    },
  ];

  return (
    <div className="min-h-screen bg-background">
      {/* Hero Section */}
      <section className="relative px-4 pt-32 pb-20 overflow-hidden">
        <div className="container mx-auto max-w-6xl">
          <HeroSection
            subtitle="AI-Powered Token Launchpad"
            title="Crypto Launchpad"
            description="Deploy tokens and manage Uniswap liquidity with natural language. An MCP server that brings blockchain operations to your AI assistant."
          />
        </div>
      </section>

      {/* Download Section */}
      <DownloadSection
        version={releaseInfo?.version}
        downloadUrl={releaseInfo?.downloadUrl}
      />

      {/* Features Section */}
      <FeaturesGrid features={features} />

      {/* Installation Section */}
      <QuickStartSection />

      {/* CTA Section */}
      <CTASection />

      {/* Footer */}
      <footer className="border-t py-8">
        <div className="container mx-auto px-4">
          <div className="flex flex-col sm:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              © 2024 Crypto Launchpad MCP. MIT License.
            </p>
            <div className="flex gap-6">
              <a
                href="https://github.com/rxtech-lab/crypto-launchpad-mcp"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                GitHub
              </a>
              <a
                href="https://github.com/rxtech-lab/crypto-launchpad-mcp/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Releases
              </a>
              <a
                href="https://github.com/rxtech-lab/crypto-launchpad-mcp/issues"
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                Support
              </a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
