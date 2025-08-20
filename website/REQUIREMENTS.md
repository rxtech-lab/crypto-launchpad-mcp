# Crypto Launchpad MCP Website Requirements

## Project Overview
A minimalist, Apple-inspired landing page for the Crypto Launchpad MCP tool with smooth animations and modern design.

## Tech Stack
- **Framework**: Next.js 15.5 with App Router
- **Styling**: Tailwind CSS v4
- **Animations**: Framer Motion
- **UI Components**: shadcn/ui (Button, Card, Badge)
- **Language**: TypeScript
- **Deployment**: Vercel (recommended)

## Design Requirements

### Visual Style
- **Apple-like Design**: Clean, minimalist aesthetic with focus on typography
- **Color Palette**: 
  - Primary: Black/Dark gray for text
  - Background: White/Light gray
  - Accent: Subtle blue for CTAs
  - Dark mode support with smooth transitions
- **Typography**: Large, bold headings with generous whitespace
- **Layout**: Single page with smooth scroll sections

### Animations
- **Scroll-triggered**: Fade-in animations as user scrolls
- **Stagger effects**: Sequential animations for list items
- **Hover interactions**: Spring-based physics for natural feel
- **Loading states**: Skeleton loaders for dynamic content
- **Page transitions**: Smooth entrance animations

## Core Features

### 1. Hero Section
- Large, bold headline introducing Launchpad MCP
- Subtle tagline with product description
- Animated entrance with fade and slide effects
- Minimal, focused messaging

### 2. Download Section
- **Dynamic Version Badge**: Fetched from GitHub releases
- **Platform Detection**: Auto-detect user's OS
- **Download Button**: 
  - Prominent CTA with hover effects
  - Shows latest version number
  - Direct link to .pkg installer for macOS
- **Alternative Downloads**: Link to all releases

### 3. Features Grid
- **Minimal Cards**: 3-4 key features
- **Icons**: Simple, elegant icons (Lucide)
- **Animations**: Stagger reveal on scroll
- **Content**: Brief, impactful descriptions

### 4. Quick Start Section
- **Code Blocks**: Syntax-highlighted installation commands
- **Copy Button**: One-click copy functionality
- **Steps**: Clear, numbered instructions
- **MCP Configuration**: Example JSON snippet

### 5. Footer
- **Links**: GitHub, Documentation, Support
- **Copyright**: Simple, unobtrusive
- **Social**: Optional social media icons

## Technical Requirements

### GitHub Integration
- **API Endpoint**: `https://api.github.com/repos/rxtech-lab/crypto-launchpad-mcp/releases/latest`
- **Caching**: 60-minute cache using Next.js `unstable_cache`
- **Error Handling**: Graceful fallback if API fails
- **Download URL Construction**: 
  ```
  https://github.com/rxtech-lab/crypto-launchpad-mcp/releases/download/{version}/launchpad-mcp_macOS_arm64_{version}.pkg
  ```

### Performance
- **Server Components**: Use RSC for initial data fetch
- **Image Optimization**: Next.js Image component
- **Font Loading**: Optimized Google Fonts (Geist)
- **Bundle Size**: Keep JavaScript minimal
- **Core Web Vitals**: Target green scores

### Accessibility
- **ARIA Labels**: Proper labeling for screen readers
- **Keyboard Navigation**: Full keyboard support
- **Focus Management**: Clear focus indicators
- **Contrast Ratios**: WCAG AA compliance
- **Reduced Motion**: Respect user preferences

## Component Structure

```
src/
├── app/
│   ├── page.tsx          # Main landing page
│   ├── layout.tsx        # Root layout with metadata
│   └── globals.css       # Global styles
├── components/
│   ├── ui/               # shadcn components
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   └── badge.tsx
│   ├── hero-section.tsx
│   ├── download-button.tsx
│   ├── feature-card.tsx
│   └── animated-container.tsx
└── lib/
    ├── github.ts         # GitHub API service
    └── utils.ts          # Utility functions
```

## Implementation Priorities

1. **Phase 1**: Core Structure
   - Set up GitHub API service with caching
   - Create basic page layout
   - Implement hero and download sections

2. **Phase 2**: Animations
   - Add Framer Motion animations
   - Implement scroll-triggered reveals
   - Add hover interactions

3. **Phase 3**: Polish
   - Fine-tune animations timing
   - Add loading states
   - Optimize performance
   - Test across devices

## SEO & Metadata
- **Title**: "Crypto Launchpad MCP - AI-Powered Token Deployment Tool"
- **Description**: "Deploy tokens and manage Uniswap liquidity with AI assistance. A Model Context Protocol server for cryptocurrency operations."
- **OG Image**: Generate social preview image
- **Keywords**: MCP, crypto, token deployment, Uniswap, AI tools

## Browser Support
- Chrome/Edge (latest 2 versions)
- Safari (latest 2 versions)
- Firefox (latest 2 versions)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Testing Checklist
- [ ] Download button fetches correct version
- [ ] All animations work smoothly
- [ ] Dark mode toggle functions
- [ ] Copy-to-clipboard works
- [ ] All links are functional
- [ ] Mobile responsive design
- [ ] Accessibility audit passes
- [ ] Performance metrics are green

## Future Enhancements
- Add testimonials section
- Include demo video/GIF
- Add changelog modal
- Multi-platform download options
- Analytics integration
- Newsletter signup