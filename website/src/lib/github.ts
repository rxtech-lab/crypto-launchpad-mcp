import { unstable_cache } from "next/cache";

interface GitHubRelease {
  tag_name: string;
  name: string;
  published_at: string;
  assets: Array<{
    name: string;
    browser_download_url: string;
  }>;
}

interface ReleaseInfo {
  version: string;
  downloadUrl: string;
  publishedAt: string;
}

async function fetchLatestRelease(): Promise<ReleaseInfo | null> {
  try {
    const response = await fetch(
      "https://api.github.com/repos/rxtech-lab/crypto-launchpad-mcp/releases/latest",
      {
        headers: {
          Accept: "application/vnd.github.v3+json",
        },
      }
    );

    if (!response.ok) {
      console.error("Failed to fetch latest release:", response.statusText);
      return null;
    }

    const release: GitHubRelease = await response.json();
    const version = release.tag_name;

    // Construct the download URL for macOS ARM64
    const downloadUrl = `https://github.com/rxtech-lab/crypto-launchpad-mcp/releases/download/${version}/launchpad-mcp_macOS_arm64_${version}.pkg`;

    return {
      version,
      downloadUrl,
      publishedAt: release.published_at,
    };
  } catch (error) {
    console.error("Error fetching latest release:", error);
    return null;
  }
}

// Cache the release info for 60 minutes (3600 seconds)
export const getLatestRelease = unstable_cache(
  fetchLatestRelease,
  ["latest-release"],
  {
    revalidate: 3600, // 60 minutes in seconds
    tags: ["github-release"],
  }
);

// Helper function to get platform-specific download URL
export function getPlatformDownloadUrl(
  version: string,
  platform: "macOS" | "linux" | "windows",
  arch: "arm64" | "amd64" = "arm64"
): string {
  const extension =
    platform === "macOS" ? "pkg" : platform === "windows" ? "exe" : "tar.gz";
  return `https://github.com/rxtech-lab/crypto-launchpad-mcp/releases/download/${version}/launchpad-mcp_${platform}_${arch}_${version}.${extension}`;
}
