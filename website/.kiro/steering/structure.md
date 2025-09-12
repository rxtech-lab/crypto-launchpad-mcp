# Project Structure & Architecture

## Directory Organization

```
src/
├── app/                    # Next.js App Router
│   ├── page.tsx           # Main landing page (Server Component)
│   ├── layout.tsx         # Root layout with metadata
│   ├── globals.css        # Global Tailwind styles
│   └── favicon.ico        # Site favicon
├── components/            # React components
│   ├── ui/               # shadcn/ui components
│   │   ├── button.tsx    # Button with CVA variants
│   │   ├── card.tsx      # Card component
│   │   ├── badge.tsx     # Badge component
│   │   ├── code-block.tsx # Syntax highlighted code
│   │   ├── tabs.tsx      # Tab navigation
│   │   ├── ide-dropdown.tsx # IDE selector
│   │   └── installation-step.tsx # Installation steps
│   ├── hero-section.tsx   # Animated hero section
│   ├── download-button.tsx # GitHub release integration
│   ├── feature-card.tsx   # Feature showcase cards
│   ├── features-grid.tsx  # Features section layout
│   ├── quick-start-section.tsx # Installation guide
│   ├── cta-section.tsx    # Call-to-action section
│   └── animated-container.tsx # Framer Motion wrapper
├── config/               # Configuration files
│   ├── ide-configs.ts    # IDE/editor configurations
│   └── installation-steps.ts # Installation instructions
├── lib/                  # Utility functions
│   ├── github.ts         # GitHub API service
│   └── utils.ts          # cn() utility function
└── types/                # TypeScript type definitions
    └── quick-start.ts    # Quick start related types
```

## Component Architecture

### Server Components (Default)

- `app/page.tsx` - Main page with data fetching
- `app/layout.tsx` - Root layout and metadata

### Client Components ("use client")

- Animation components using Framer Motion
- Interactive components with state/events
- Components using browser APIs

## Naming Conventions

### Files & Directories

- `kebab-case` for file names
- `PascalCase` for component files
- `camelCase` for utility functions

### Components

- `PascalCase` for component names
- Props interfaces: `ComponentNameProps`
- Variants: Use CVA with descriptive variant names

### CSS Classes

- Tailwind utility classes
- CSS variables for theming: `--color-*`
- Component-specific classes: `component-name__element`

## Data Flow Patterns

### GitHub Integration

- Server-side data fetching in `page.tsx`
- Cached API calls using `unstable_cache`
- Props passed down to client components

### Animation Patterns

- Framer Motion with consistent easing: `[0.21, 0.47, 0.32, 0.98]`
- Stagger animations for lists
- Scroll-triggered reveals
- Spring physics for interactions

## Configuration Files

### Root Level

- `components.json` - shadcn/ui configuration
- `biome.json` - Linting and formatting rules
- `next.config.ts` - Next.js configuration
- `tsconfig.json` - TypeScript configuration
- `postcss.config.mjs` - PostCSS for Tailwind
