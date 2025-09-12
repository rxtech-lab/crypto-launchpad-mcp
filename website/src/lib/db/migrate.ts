import { migrate } from "drizzle-orm/postgres-js/migrator";
import { db, client, checkDatabaseConnection } from "./connection";

/**
 * Run database migrations
 * This function applies all pending migrations to the database
 */
export async function runMigrations(): Promise<void> {
  try {
    console.log("Checking database connection...");
    const isConnected = await checkDatabaseConnection();

    if (!isConnected) {
      throw new Error("Cannot connect to database");
    }

    console.log("Running database migrations...");
    await migrate(db, { migrationsFolder: "./drizzle" });
    console.log("Migrations completed successfully");
  } catch (error) {
    console.error("Migration failed:", error);
    throw error;
  }
}

/**
 * Initialize database with migrations
 * Use this for setting up a fresh database
 */
export async function initializeDatabase(): Promise<void> {
  try {
    console.log("Initializing database...");
    await runMigrations();
    console.log("Database initialized successfully");
  } catch (error) {
    console.error("Database initialization failed:", error);
    throw error;
  } finally {
    // Close the connection after migration
    await client.end();
  }
}

// CLI script for running migrations
// Only run when executed directly, not when imported
if (typeof require !== "undefined" && require.main === module) {
  initializeDatabase()
    .then(() => {
      console.log("Migration script completed");
      if (typeof process !== "undefined") {
        process.exit(0);
      }
    })
    .catch((error) => {
      console.error("Migration script failed:", error);
      if (typeof process !== "undefined") {
        process.exit(1);
      }
    });
}
