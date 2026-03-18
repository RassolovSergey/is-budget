import { redirect } from "next/navigation";

export default function PlannedExpensesRedirect({ params }: { params: { id: string } }) {
  redirect(`/projects/${params.id}/actual-expenses`);
}
