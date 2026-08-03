import type { Metadata } from "next";
import { ReceiptVerification } from "@/components/receipt-verification";

export const metadata: Metadata = {
  title: "Verificar comprobante | Control Propiedades",
  robots: { index: false, follow: false },
};

export default async function VerifyReceiptPage({ params }: { params: Promise<{ token: string }> }) {
  const { token } = await params;
  return <ReceiptVerification token={token} />;
}
