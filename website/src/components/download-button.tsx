"use client";

import { motion } from "framer-motion";
import { Download, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface DownloadButtonProps {
  version?: string;
  downloadUrl?: string;
  loading?: boolean;
  className?: string;
}

export function DownloadButton({
  version,
  downloadUrl,
  loading = false,
  className,
}: DownloadButtonProps) {
  const handleDownload = () => {
    if (downloadUrl) {
      window.open(downloadUrl, "_blank");
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{
        duration: 0.6,
        delay: 0.4,
        ease: [0.21, 0.47, 0.32, 0.98],
      }}
      className={cn("flex flex-col items-center gap-3", className)}
    >
      <motion.div
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        transition={{ type: "spring", stiffness: 400, damping: 17 }}
        className="relative"
      >
        <Button
          size="lg"
          onClick={handleDownload}
          disabled={loading || !downloadUrl}
          className="h-14 px-8 text-base font-semibold shadow-lg hover:shadow-xl transition-shadow duration-300 hover:cursor-pointer"
        >
          {loading ? (
            <>
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
              Loading...
            </>
          ) : (
            <>
              <Download className="mr-2 h-5 w-5" />
              Download for macOS
            </>
          )}
        </Button>

        {version && (
          <motion.div
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.4, delay: 0.6 }}
            className="absolute -top-4 -right-2"
          >
            <Badge
              variant="default"
              className="px-2 py-1 text-xs font-medium border-2 bg-muted text-black"
            >
              {version}
            </Badge>
          </motion.div>
        )}
      </motion.div>

      <motion.p
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.6, delay: 0.7 }}
        className="text-sm text-muted-foreground"
      >
        Requires macOS 11.0 or later • ARM64
      </motion.p>
    </motion.div>
  );
}

export function DownloadSection({
  version,
  downloadUrl,
  loading = false,
}: DownloadButtonProps) {
  return (
    <motion.section
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.8, delay: 0.2 }}
      className="relative pb-20"
    >
      <div className="container mx-auto px-4">
        <div className="flex flex-col items-center text-center">
          <DownloadButton
            version={version}
            downloadUrl={downloadUrl}
            loading={loading}
          />

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6, delay: 0.8 }}
            className="mt-8 flex items-center gap-6 text-sm text-muted-foreground"
          >
            <a
              href="https://github.com/rxtech-lab/crypto-launchpad-mcp/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-foreground transition-colors"
            >
              View all releases →
            </a>
            <span className="text-muted-foreground/50">•</span>
            <a
              href="https://github.com/rxtech-lab/crypto-launchpad-mcp"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-foreground transition-colors"
            >
              Source code →
            </a>
          </motion.div>
        </div>
      </div>

      {/* Subtle pulse animation for attention */}
      <motion.div
        className="absolute inset-0 -z-10 pointer-events-none"
        initial={{ opacity: 0 }}
        animate={{ opacity: [0, 0.5, 0] }}
        transition={{
          duration: 3,
          repeat: Infinity,
          repeatDelay: 2,
          ease: "easeInOut",
        }}
      >
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-primary/10 rounded-full blur-3xl" />
      </motion.div>
    </motion.section>
  );
}
