import { User } from "next-auth";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";

interface UserProfileProps {
  user: User;
}

export function UserProfile({ user }: UserProfileProps) {
  // Get user initials for avatar fallback
  const getInitials = (name?: string | null, email?: string | null) => {
    if (name) {
      return name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2);
    }
    if (email) {
      return email[0].toUpperCase();
    }
    return "U";
  };

  return (
    <Card className="p-6">
      <div className="flex items-center space-x-4">
        {/* User Avatar */}
        <Avatar className="h-16 w-16">
          <AvatarImage
            src={user.image || undefined}
            alt={user.name || "User"}
          />
          <AvatarFallback className="text-lg font-semibold">
            {getInitials(user.name, user.email)}
          </AvatarFallback>
        </Avatar>

        {/* User Information */}
        <div className="flex-1">
          <div className="flex items-center space-x-3 mb-2">
            <h2 className="text-2xl font-bold text-gray-900">
              {user.name || "Anonymous User"}
            </h2>
            <Badge variant="secondary" className="text-xs">
              Authenticated
            </Badge>
          </div>

          {user.email && <p className="text-gray-600 mb-2">{user.email}</p>}

          <div className="flex items-center space-x-4 text-sm text-gray-500">
            <span>User ID: {user.id}</span>
          </div>
        </div>
      </div>
    </Card>
  );
}
