import {
  describe,
  it,
  expect,
  beforeEach,
  vi,
  type MockedFunction,
} from "vitest";
import { eq, and, desc } from "drizzle-orm";
import * as queries from "../queries";
import { db } from "../connection";
import {
  users,
  sessions,
  jwtTokens,
  authenticators,
  type User,
  type NewUser,
  type Session,
  type NewSession,
  type JwtToken,
  type NewJwtToken,
  type Authenticator,
  type NewAuthenticator,
} from "../schema";

// Mock the database connection
vi.mock("../connection", () => ({
  db: {
    insert: vi.fn(),
    select: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockDb = db as {
  insert: MockedFunction<any>;
  select: MockedFunction<any>;
  update: MockedFunction<any>;
  delete: MockedFunction<any>;
};

describe("Database Queries", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("User queries", () => {
    const mockUser: User = {
      id: "user-123",
      email: "test@example.com",
      name: "Test User",
      image: "https://example.com/avatar.jpg",
      createdAt: new Date(),
      updatedAt: new Date(),
    };

    const mockNewUser: NewUser = {
      id: "user-123",
      email: "test@example.com",
      name: "Test User",
      image: "https://example.com/avatar.jpg",
    };

    describe("createUser", () => {
      it("should create a user successfully", async () => {
        const mockInsert = {
          values: vi.fn().mockReturnThis(),
          returning: vi.fn().mockResolvedValue([mockUser]),
        };
        mockDb.insert.mockReturnValue(mockInsert);

        const result = await queries.createUser(mockNewUser);

        expect(mockDb.insert).toHaveBeenCalledWith(users);
        expect(mockInsert.values).toHaveBeenCalledWith(mockNewUser);
        expect(mockInsert.returning).toHaveBeenCalled();
        expect(result).toEqual(mockUser);
      });

      it("should throw error when creation fails", async () => {
        const mockInsert = {
          values: vi.fn().mockReturnThis(),
          returning: vi.fn().mockRejectedValue(new Error("Database error")),
        };
        mockDb.insert.mockReturnValue(mockInsert);

        await expect(queries.createUser(mockNewUser)).rejects.toThrow(
          "Failed to create user"
        );
      });
    });

    describe("getUserById", () => {
      it("should return user when found", async () => {
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue([mockUser]),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getUserById("user-123");

        expect(mockDb.select).toHaveBeenCalled();
        expect(mockSelect.from).toHaveBeenCalledWith(users);
        expect(mockSelect.where).toHaveBeenCalledWith(eq(users.id, "user-123"));
        expect(result).toEqual(mockUser);
      });

      it("should return null when user not found", async () => {
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue([]),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getUserById("nonexistent");

        expect(result).toBeNull();
      });

      it("should throw error when query fails", async () => {
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockRejectedValue(new Error("Database error")),
        };
        mockDb.select.mockReturnValue(mockSelect);

        await expect(queries.getUserById("user-123")).rejects.toThrow(
          "Failed to fetch user"
        );
      });
    });

    describe("getUserByEmail", () => {
      it("should return user when found by email", async () => {
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue([mockUser]),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getUserByEmail("test@example.com");

        expect(mockSelect.where).toHaveBeenCalledWith(
          eq(users.email, "test@example.com")
        );
        expect(result).toEqual(mockUser);
      });
    });
  });

  describe("Session queries", () => {
    const mockSession: Session = {
      sessionToken: "session-123",
      userId: "user-123",
      expires: new Date(),
    };

    const mockNewSession: NewSession = {
      sessionToken: "session-123",
      userId: "user-123",
      expires: new Date(),
    };

    describe("createSession", () => {
      it("should create a session successfully", async () => {
        const mockInsert = {
          values: vi.fn().mockReturnThis(),
          returning: vi.fn().mockResolvedValue([mockSession]),
        };
        mockDb.insert.mockReturnValue(mockInsert);

        const result = await queries.createSession(mockNewSession);

        expect(mockDb.insert).toHaveBeenCalledWith(sessions);
        expect(result).toEqual(mockSession);
      });
    });

    describe("getSessionByToken", () => {
      it("should return session when found", async () => {
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue([mockSession]),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getSessionByToken("session-123");

        expect(mockSelect.where).toHaveBeenCalledWith(
          eq(sessions.sessionToken, "session-123")
        );
        expect(result).toEqual(mockSession);
      });
    });

    describe("getUserSessions", () => {
      it("should return user sessions ordered by expiration", async () => {
        const mockSessions = [mockSession];
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockReturnThis(),
          orderBy: vi.fn().mockResolvedValue(mockSessions),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getUserSessions("user-123");

        expect(mockSelect.where).toHaveBeenCalledWith(
          eq(sessions.userId, "user-123")
        );
        expect(mockSelect.orderBy).toHaveBeenCalledWith(desc(sessions.expires));
        expect(result).toEqual(mockSessions);
      });
    });

    describe("deleteSession", () => {
      it("should delete session successfully", async () => {
        const mockDelete = {
          where: vi.fn().mockResolvedValue(undefined),
        };
        mockDb.delete.mockReturnValue(mockDelete);

        await queries.deleteSession("session-123");

        expect(mockDb.delete).toHaveBeenCalledWith(sessions);
        expect(mockDelete.where).toHaveBeenCalledWith(
          eq(sessions.sessionToken, "session-123")
        );
      });
    });
  });

  describe("JWT Token queries", () => {
    const mockJwtToken: JwtToken = {
      id: "token-123",
      userId: "user-123",
      tokenName: "Test Token",
      jti: "jti-123",
      aud: ["audience"],
      clientId: "client-123",
      roles: ["user"],
      scopes: ["read"],
      createdAt: new Date(),
      expiresAt: new Date(),
      isActive: true,
    };

    const mockNewJwtToken: NewJwtToken = {
      id: "token-123",
      userId: "user-123",
      tokenName: "Test Token",
      jti: "jti-123",
      aud: ["audience"],
      clientId: "client-123",
      roles: ["user"],
      scopes: ["read"],
    };

    describe("createJwtToken", () => {
      it("should create JWT token successfully", async () => {
        const mockInsert = {
          values: vi.fn().mockReturnThis(),
          returning: vi.fn().mockResolvedValue([mockJwtToken]),
        };
        mockDb.insert.mockReturnValue(mockInsert);

        const result = await queries.createJwtToken(mockNewJwtToken);

        expect(mockDb.insert).toHaveBeenCalledWith(jwtTokens);
        expect(result).toEqual(mockJwtToken);
      });
    });

    describe("getUserJwtTokens", () => {
      it("should return active user tokens", async () => {
        const mockTokens = [mockJwtToken];
        const mockSelect = {
          from: vi.fn().mockReturnThis(),
          where: vi.fn().mockReturnThis(),
          orderBy: vi.fn().mockResolvedValue(mockTokens),
        };
        mockDb.select.mockReturnValue(mockSelect);

        const result = await queries.getUserJwtTokens("user-123");

        expect(mockSelect.where).toHaveBeenCalledWith(
          and(eq(jwtTokens.userId, "user-123"), eq(jwtTokens.isActive, true))
        );
        expect(result).toEqual(mockTokens);
      });
    });

    describe("deactivateJwtToken", () => {
      it("should deactivate token successfully", async () => {
        const mockUpdate = {
          set: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue(undefined),
        };
        mockDb.update.mockReturnValue(mockUpdate);

        await queries.deactivateJwtToken("token-123");

        expect(mockDb.update).toHaveBeenCalledWith(jwtTokens);
        expect(mockUpdate.set).toHaveBeenCalledWith({ isActive: false });
        expect(mockUpdate.where).toHaveBeenCalledWith(
          eq(jwtTokens.id, "token-123")
        );
      });
    });
  });

  describe("Authenticator queries", () => {
    const mockAuthenticator: Authenticator = {
      credentialID: "cred-123",
      userId: "user-123",
      providerAccountId: "provider-123",
      credentialPublicKey: "public-key",
      counter: 0,
      credentialDeviceType: "singleDevice",
      credentialBackedUp: false,
      transports: "usb,nfc",
    };

    describe("createAuthenticator", () => {
      it("should create authenticator successfully", async () => {
        const mockInsert = {
          values: vi.fn().mockReturnThis(),
          returning: vi.fn().mockResolvedValue([mockAuthenticator]),
        };
        mockDb.insert.mockReturnValue(mockInsert);

        const result = await queries.createAuthenticator(mockAuthenticator);

        expect(mockDb.insert).toHaveBeenCalledWith(authenticators);
        expect(result).toEqual(mockAuthenticator);
      });
    });

    describe("updateAuthenticatorCounter", () => {
      it("should update counter successfully", async () => {
        const mockUpdate = {
          set: vi.fn().mockReturnThis(),
          where: vi.fn().mockResolvedValue(undefined),
        };
        mockDb.update.mockReturnValue(mockUpdate);

        await queries.updateAuthenticatorCounter("cred-123", 5);

        expect(mockUpdate.set).toHaveBeenCalledWith({ counter: 5 });
        expect(mockUpdate.where).toHaveBeenCalledWith(
          eq(authenticators.credentialID, "cred-123")
        );
      });
    });
  });
});
