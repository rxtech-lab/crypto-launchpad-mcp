"use client"

import { FadeIn } from "@/components/animated-container"
import { GitBranch, Terminal } from "lucide-react"

export function CTASection() {
  return (
    <section className="py-20">
      <div className="container mx-auto px-4 max-w-4xl">
        <FadeIn className="text-center">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Ready to Launch?
          </h2>
          <p className="text-lg text-muted-foreground mb-8">
            Join the future of AI-powered blockchain development
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            <a
              href="https://github.com/rxtech-lab/crypto-launchpad-mcp"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium h-11 px-8 bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <GitBranch className="h-4 w-4" />
              View on GitHub
            </a>
            <a
              href="https://github.com/rxtech-lab/crypto-launchpad-mcp/blob/main/README.md"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center justify-center gap-2 rounded-md text-sm font-medium h-11 px-8 border border-input bg-background hover:bg-accent hover:text-accent-foreground transition-colors"
            >
              <Terminal className="h-4 w-4" />
              Documentation
            </a>
          </div>
        </FadeIn>
      </div>
    </section>
  )
}