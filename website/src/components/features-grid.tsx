"use client"

import { motion } from "framer-motion"
import { 
  Rocket, 
  Shield, 
  Zap, 
  Code2, 
  Coins, 
  GitBranch,
  LucideIcon
} from "lucide-react"
import { FeatureCard } from "./feature-card"

const iconMap: Record<string, LucideIcon> = {
  Rocket,
  Shield,
  Zap,
  Code2,
  Coins,
  GitBranch,
}

interface Feature {
  iconName: string
  title: string
  description: string
}

interface FeaturesGridProps {
  features: Feature[]
}

export function FeaturesGrid({ features }: FeaturesGridProps) {
  return (
    <motion.section
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.8 }}
      className="py-20"
    >
      <div className="container mx-auto px-4">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="text-center mb-12"
        >
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Powerful Features
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Everything you need to deploy tokens and manage liquidity with AI assistance
          </p>
        </motion.div>

        <motion.div
          initial="hidden"
          animate="visible"
          variants={{
            hidden: { opacity: 0 },
            visible: {
              opacity: 1,
              transition: {
                staggerChildren: 0.1,
              },
            },
          }}
          className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
        >
          {features.map((feature, index) => {
            const Icon = iconMap[feature.iconName]
            return (
              <motion.div
                key={index}
                variants={{
                  hidden: { opacity: 0, y: 20 },
                  visible: {
                    opacity: 1,
                    y: 0,
                    transition: {
                      duration: 0.5,
                      ease: [0.21, 0.47, 0.32, 0.98],
                    },
                  },
                }}
              >
                <FeatureCard 
                  icon={Icon}
                  title={feature.title}
                  description={feature.description}
                />
              </motion.div>
            )
          })}
        </motion.div>
      </div>
    </motion.section>
  )
}