import { Navigate, Route, Routes } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { AppShell } from "@/components/app-shell";
import { LoginPage } from "@/pages/login";
import { DashboardPage } from "@/pages/dashboard";
import { TicketsListPage } from "@/pages/tickets-list";
import { TicketDetailPage } from "@/pages/ticket-detail";
import { CustomersListPage } from "@/pages/customers-list";
import { CustomerDetailPage } from "@/pages/customer-detail";
import { UsersListPage } from "@/pages/users-list";
import { UserDetailPage } from "@/pages/user-detail";
import { KnowledgeListPage } from "@/pages/knowledge-list";
import { KnowledgeDetailPage } from "@/pages/knowledge-detail";
import { AccessPage } from "@/pages/access";
import { ProductsListPage } from "@/pages/products-list";
import { ProductDetailPage } from "@/pages/product-detail";
import { ServicesListPage } from "@/pages/services-list";
import { ServiceDetailPage } from "@/pages/service-detail";
import { SLAPage } from "@/pages/sla";
import { DataJobsPage } from "@/pages/data-jobs";
import { LLMProvidersPage } from "@/pages/llm-providers";
import { SubscriptionsPage } from "@/pages/subscriptions-list";
import type { JSX } from "react";

function FullScreenLoader() {
  return (
    <div className="grid min-h-screen place-items-center">
      <Loader2 className="size-6 animate-spin text-primary" />
    </div>
  );
}

function Protected({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth();
  if (loading) return <FullScreenLoader />;
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

/**
 * TeamOnly gates the operator-facing admin areas. Customer-role users are
 * redirected to their tickets (the backend would 403 these endpoints anyway).
 */
function TeamOnly({ children }: { children: JSX.Element }) {
  const { user } = useAuth();
  if (user?.role === "customer") return <Navigate to="/tickets" replace />;
  return children;
}

function PublicOnly({ children }: { children: JSX.Element }) {
  const { user, loading } = useAuth();
  if (loading) return <FullScreenLoader />;
  if (user) return <Navigate to="/dashboard" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <PublicOnly>
            <LoginPage />
          </PublicOnly>
        }
      />
      <Route
        element={
          <Protected>
            <AppShell />
          </Protected>
        }
      >
        {/* Available to every authenticated user (customer data is scoped
            server-side by the Actor). */}
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/tickets" element={<TicketsListPage />} />
        <Route path="/tickets/:id" element={<TicketDetailPage />} />
        <Route path="/knowledge" element={<KnowledgeListPage />} />
        <Route path="/knowledge/:id" element={<KnowledgeDetailPage />} />

        {/* Team-only operator areas. */}
        <Route path="/customers" element={<TeamOnly><CustomersListPage /></TeamOnly>} />
        <Route path="/customers/:id" element={<TeamOnly><CustomerDetailPage /></TeamOnly>} />
        <Route path="/users" element={<TeamOnly><UsersListPage /></TeamOnly>} />
        <Route path="/users/:id" element={<TeamOnly><UserDetailPage /></TeamOnly>} />
        <Route path="/products" element={<TeamOnly><ProductsListPage /></TeamOnly>} />
        <Route path="/products/:id" element={<TeamOnly><ProductDetailPage /></TeamOnly>} />
        <Route path="/services" element={<TeamOnly><ServicesListPage /></TeamOnly>} />
        <Route path="/services/:id" element={<TeamOnly><ServiceDetailPage /></TeamOnly>} />
        <Route path="/subscriptions" element={<TeamOnly><SubscriptionsPage /></TeamOnly>} />
        <Route path="/sla" element={<TeamOnly><SLAPage /></TeamOnly>} />
        <Route path="/data" element={<TeamOnly><DataJobsPage /></TeamOnly>} />
        <Route path="/rbac" element={<TeamOnly><AccessPage /></TeamOnly>} />
        <Route path="/llm" element={<TeamOnly><LLMProvidersPage /></TeamOnly>} />
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}
