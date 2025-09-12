import { Card } from "@/components/ui/card";

export default function DashboardPage() {
  return (
    <div className="space-y-6">
      {/* Dashboard Overview */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 mb-2">
          Dashboard Overview
        </h1>
        <p className="text-gray-600">
          Manage your JWT tokens and active sessions from this dashboard.
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">Active Tokens</p>
              <p className="text-2xl font-bold text-gray-900">0</p>
            </div>
          </div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">
                Active Sessions
              </p>
              <p className="text-2xl font-bold text-gray-900">1</p>
            </div>
          </div>
        </Card>

        <Card className="p-6">
          <div className="flex items-center">
            <div className="flex-1">
              <p className="text-sm font-medium text-gray-600">Last Login</p>
              <p className="text-2xl font-bold text-gray-900">Today</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Quick Actions */}
      <Card className="p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          Quick Actions
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <h3 className="font-medium text-gray-900 mb-2">Create JWT Token</h3>
            <p className="text-sm text-gray-600 mb-3">
              Generate a new JWT token for API access
            </p>
            <a
              href="/dashboard/tokens"
              className="text-sm font-medium text-blue-600 hover:text-blue-500"
            >
              Go to Tokens →
            </a>
          </div>

          <div className="p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <h3 className="font-medium text-gray-900 mb-2">Manage Sessions</h3>
            <p className="text-sm text-gray-600 mb-3">
              View and manage your active sessions
            </p>
            <a
              href="/dashboard/sessions"
              className="text-sm font-medium text-blue-600 hover:text-blue-500"
            >
              Go to Sessions →
            </a>
          </div>
        </div>
      </Card>
    </div>
  );
}
