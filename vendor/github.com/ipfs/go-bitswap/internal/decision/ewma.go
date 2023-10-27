package decision

func ewma(old, new, alpha float64) float64 {
	return new*alpha + (1-alpha)*old
}
