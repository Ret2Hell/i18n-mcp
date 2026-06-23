package diff

func Summarize(items []KeyDiff) Summary {
	summary := Summary{ByLocale: map[string]StatusCounts{}}
	for _, item := range items {
		summary.Total++
		addStatus(&summary, item.Status)
		counts := summary.ByLocale[item.Locale]
		addStatusCounts(&counts, item.Status)
		summary.ByLocale[item.Locale] = counts
	}
	if len(summary.ByLocale) == 0 {
		summary.ByLocale = nil
	}
	return summary
}

func addStatus(summary *Summary, status KeyStatus) {
	switch status {
	case Current:
		summary.Current++
	case Missing:
		summary.Missing++
	case Stale:
		summary.Stale++
	case Extra:
		summary.Extra++
	case Invalid:
		summary.Invalid++
	case Unknown:
		summary.Unknown++
	}
}

func addStatusCounts(counts *StatusCounts, status KeyStatus) {
	switch status {
	case Current:
		counts.Current++
	case Missing:
		counts.Missing++
	case Stale:
		counts.Stale++
	case Extra:
		counts.Extra++
	case Invalid:
		counts.Invalid++
	case Unknown:
		counts.Unknown++
	}
}
