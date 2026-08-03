import { DocumentReview } from "@/components/document-review";

export default async function DocumentReviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <DocumentReview id={id} />;
}
