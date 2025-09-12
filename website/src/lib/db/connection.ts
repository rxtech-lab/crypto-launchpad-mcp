import { drizzle } from "drizzle-orm/postgres-js";
import * as postgres from "postgres";
import * as schema from "./schema";

if (!process.env.DATABASE_URL) {
  throw new Error("DATABASE_URL environment variable is required");
}

// Create the connection with proper configuration
const connectionString = process.env.DATABASE_URL;

// Configure postgres client with connection pooling and error handling
const client = postgres.default(connectionString, {
  prepare: false,
  max: 10, // Maximum number of connections in the pool
  idle_timeout: 20, // Close idle connections after 20 seconds
  connect_timeout: 10, // Connection timeout in seconds
  onnotice: () => {}, // Suppress notices in production
});

// Create the database instance with schema
export const db = drizzle(client, { schema });

// Connection health check function
export async function checkDatabaseConnection(): Promise<boolean> {
  try {
    await client`SELECT 1`;
    return true;
  } catch (error) {
    console.error("Database connection failed:", error);
    return false;
  }
}

// Graceful shutdown function
export async function closeDatabaseConnection(): Promise<void> {
  try {
    await client.end();
    console.log("Database connection closed gracefully");
  } catch (error) {
    console.error("Error closing database connection:", error);
  }
}

// Export the client for direct queries if needed
export { client };
