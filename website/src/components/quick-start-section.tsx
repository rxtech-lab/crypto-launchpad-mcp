"use client";

import {
  AnimatedContainer,
  StaggerContainer,
  StaggerItem,
} from "@/components/animated-container";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Check, Terminal, Code2, FileCode2, Blocks } from "lucide-react";

export function QuickStartSection() {
  return (
    <section className="py-20 bg-muted/30">
      <div className="container mx-auto px-4 max-w-4xl">
        <AnimatedContainer className="text-center mb-12">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">Quick Start</h2>
          <p className="text-lg text-muted-foreground">
            Get up and running in minutes
          </p>
        </AnimatedContainer>

        <StaggerContainer className="space-y-8">
          <StaggerItem>
            <div className="bg-background rounded-xl border p-6 shadow-sm">
              <div className="flex items-start gap-4">
                <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <span className="text-sm font-semibold text-primary">1</span>
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold mb-2">Download and Install</h3>
                  <p className="text-muted-foreground mb-4">
                    Download the latest release for macOS and run the installer.
                  </p>
                  <div className="bg-muted/50 rounded-lg p-4 font-mono text-sm">
                    # The binary will be installed to
                    /usr/local/bin/launchpad-mcp
                  </div>
                </div>
              </div>
            </div>
          </StaggerItem>

          <StaggerItem>
            <div className="bg-background rounded-xl border p-6 shadow-sm">
              <div className="flex items-start gap-4">
                <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <span className="text-sm font-semibold text-primary">2</span>
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold mb-2">Configure Your Client</h3>
                  <p className="text-muted-foreground mb-4">
                    Add the MCP server to your preferred AI client
                    configuration.
                  </p>

                  <Tabs defaultValue="claude" className="w-full">
                    <TabsList className="grid w-full grid-cols-2">
                      <TabsTrigger value="claude">
                        <Terminal className="mr-1.5 h-3.5 w-3.5" />
                        Claude
                      </TabsTrigger>

                      <TabsTrigger value="cursor">
                        <FileCode2 className="mr-1.5 h-3.5 w-3.5" />
                        Cursor
                      </TabsTrigger>
                    </TabsList>

                    <TabsContent value="claude" className="mt-4">
                      <div className="space-y-3">
                        <p className="text-sm text-muted-foreground">
                          Add to your Claude Desktop configuration file:
                        </p>
                        <div className="bg-muted/50 rounded-lg p-4 font-mono text-sm overflow-x-auto">
                          <pre>{`{
  "mcpServers": {
    "launchpad-mcp": {
      "command": "/usr/local/bin/launchpad-mcp",
      "args": []
    }
  }
}`}</pre>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Location: ~/Library/Application
                          Support/Claude/claude_desktop_config.json
                        </p>
                      </div>
                    </TabsContent>

                    <TabsContent value="cursor" className="mt-4">
                      <div className="space-y-3">
                        <p className="text-sm text-muted-foreground">
                          Add to your Cursor settings:
                        </p>
                        <div className="bg-muted/50 rounded-lg p-4 font-mono text-sm overflow-x-auto">
                          <pre>{`{
  "mcpServers": {
    "launchpad-mcp": {
      "command": "/usr/local/bin/launchpad-mcp",
      "args": [],
      "env": {}
    }
  }
}`}</pre>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          Open Settings → MCP → Add Server
                        </p>
                      </div>
                    </TabsContent>
                  </Tabs>
                </div>
              </div>
            </div>
          </StaggerItem>

          <StaggerItem>
            <div className="bg-background rounded-xl border p-6 shadow-sm">
              <div className="flex items-start gap-4">
                <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                  <span className="text-sm font-semibold text-primary">3</span>
                </div>
                <div className="flex-1">
                  <h3 className="font-semibold mb-2">Start Using</h3>
                  <p className="text-muted-foreground mb-4">
                    Restart Claude Desktop and start deploying tokens with
                    natural language.
                  </p>
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm">
                      <Check className="h-4 w-4 text-primary" />
                      <span>Deploy ERC-20 tokens on Ethereum</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Check className="h-4 w-4 text-primary" />
                      <span>Create Uniswap liquidity pools</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Check className="h-4 w-4 text-primary" />
                      <span>Manage liquidity positions</span>
                    </div>
                    <div className="flex items-center gap-2 text-sm">
                      <Check className="h-4 w-4 text-primary" />
                      <span>Execute token swaps</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </StaggerItem>
        </StaggerContainer>
      </div>
    </section>
  );
}
