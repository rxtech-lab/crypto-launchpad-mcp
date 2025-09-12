import { auth } from "@/auth";
import { redirect } from "next/navigation";
import { DashboardNav } from "@/components/dashboard/dashboard-nav";
import { UserProfile } from "@/components/dashboard/user-profile";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth();

  // Redirect unauthenticated users to auth page
  if (!session) {
    redirect("/auth");
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Dashboard Navigation */}
      <DashboardNav />

      {/* Main Content Area */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* User Profile Section */}
        <div className="mb-8">
          <UserProfile user={session.user} />
        </div>

        {/* Page Content */}
        <main>{children}</main>
      </div>
    </div>
  );
}
