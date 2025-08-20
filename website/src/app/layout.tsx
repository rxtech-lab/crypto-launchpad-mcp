import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Crypto Launchpad MCP - AI-Powered Token Deployment",
  description:
    "Deploy tokens and manage Uniswap liquidity with AI assistance. A Model Context Protocol server for cryptocurrency operations.",
  keywords:
    "MCP, crypto, token deployment, Uniswap, AI tools, blockchain, Ethereum, Solana",
  authors: [{ name: "RxTech Lab" }],
  openGraph: {
    title: "Crypto Launchpad MCP",
    description: "AI-powered token deployment and liquidity management",
    url: "https://cryptolaunch.app",
    siteName: "Crypto Launchpad",
    type: "website",
    images: [
      {
        url: "/og-image.png",
        width: 1200,
        height: 630,
        alt: "Crypto Launchpad MCP",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Crypto Launchpad MCP",
    description: "AI-powered token deployment and liquidity management",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
