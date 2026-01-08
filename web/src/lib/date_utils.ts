export function relative_date(d1: Date, d2: Date) {
  const diff = Math.abs(d2.getTime() - d1.getTime());
  const diffDays = Math.ceil(diff / (1000 * 3600 * 24));

  if (diffDays === 0) {
    return "today";
  } else if (diffDays === 1) {
    return "yesterday";
  } else if (diffDays < 7) {
    return diffDays + " days ago";
  }

  const offset = d1.getTimezoneOffset();
  d1 = new Date(d1.getTime() - offset * 60 * 1000);
  return "on " + d1.toISOString().split("T")[0];
}
