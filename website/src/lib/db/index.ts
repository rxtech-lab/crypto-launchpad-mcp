// Export database connection and utilities
export {
  db,
  checkDatabaseConnection,
  closeDatabaseConnection,
} from "./connection";

// Export schema and types
export * from "./schema";

// Export query functions
export * from "./queries";

// Note: Migration utilities are not exported here to avoid Edge Runtime issues
// Import them directly from "./migrate" when needed in Node.js environments
