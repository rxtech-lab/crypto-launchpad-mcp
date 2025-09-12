# Tech Stack & Development

## Framework & Core Technologies

- **Next.js 15.5** with App Router and React Server Components
- **React 19.1** with TypeScript 5
- **Tailwind CSS v4** for styling with CSS variables
- **Framer Motion 12** for animations and micro-interactions

## UI Components & Design System

- **shadcn/ui** components (New York style)
- **Radix UI** primitives for accessibility
- **Lucide React** for icons
- **Class Variance Authority (CVA)** for component variants
- **clsx + tailwind-merge** utility via `cn()` helper

## Code Quality & Formatting

- **Biome** for linting and formatting (replaces ESLint + Prettier)
- **TypeScript** strict mode enabled
- 2-space indentation, organized imports

## Key Libraries

- `framer-motion` - Animations with spring physics
- `class-variance-authority` - Type-safe component variants
- `tailwind-merge` - Conditional Tailwind class merging
- `@radix-ui/react-*` - Accessible UI primitives

## Common Commands

### Development

```bash
pnpm dev          # Start dev server with Turbopack
pnpm build        # Production build with Turbopack
pnpm start        # Start production server
```

### Code Quality

```bash
pnpm lint         # Run Biome linter
pnpm format       # Format code with Biome
```

## Build Configuration

- **Turbopack** enabled for faster builds and dev server
- **Path aliases**: `@/*` maps to `./src/*`
- **CSS Variables** for theming support
- **Server Components** by default, client components marked with "use client"

## Performance Optimizations

- Server-side data fetching with `unstable_cache`
- Next.js Image optimization
- Font optimization with next/font
- Bundle splitting and tree shaking

## Coding requirement

- Never run pnpm dev& to start dev server. The server is running in the background already
- No need to write test for now
