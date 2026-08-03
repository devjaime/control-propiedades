import type { Metadata } from "next";
import { TenantPortalView } from "@/components/tenant-portal-view";

export const metadata: Metadata = {
  title: "Seguimiento de arreglos | Control Propiedades",
  robots: { index: false, follow: false },
};

export default async function TenantTrackingPage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params;
  return <TenantPortalView token={token} />;
}
