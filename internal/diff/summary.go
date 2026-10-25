package diff

// Add counts one record without retaining its payload.
func (s *Summary) Add(item KeyDiff) {
	if s.ByLocale == nil {
		s.ByLocale = map[string]StatusCounts{}
	}
	s.Total++
	addStatus(s, item.Status)
	counts := s.ByLocale[item.Locale]
	addStatusCounts(&counts, item.Status)
	s.ByLocale[item.Locale] = counts
}

// Summarize counts key diff statuses across items.
func Summarize(items []KeyDiff) Summary {
	summary := Summary{ByLocale: make(map[string]StatusCounts, len(items))}
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
	counts := StatusCounts{
		SourceInvalid: summary.SourceInvalid,
		Current:       summary.Current,
		Missing:       summary.Missing,
		Stale:         summary.Stale,
		Extra:         summary.Extra,
		Invalid:       summary.Invalid,
		Unknown:       summary.Unknown,
	}
	addStatusCounts(&counts, status)
	summary.SourceInvalid = counts.SourceInvalid
	summary.Current = counts.Current
	summary.Missing = counts.Missing
	summary.Stale = counts.Stale
	summary.Extra = counts.Extra
	summary.Invalid = counts.Invalid
	summary.Unknown = counts.Unknown
}

func addStatusCounts(counts *StatusCounts, status KeyStatus) {
	switch status {
	case SourceInvalid:
		counts.SourceInvalid++
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
