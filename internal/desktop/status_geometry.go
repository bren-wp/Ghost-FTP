package desktop

const (
	statusBandHeight      = 24
	statusBandBottomInset = 10
	statusBandContentGap  = 7
)

// statusBandGeometry keeps the footer status controls visibly inside the
// client area while reserving a stable gap above them for transfer content.
// The calculation is kept platform-neutral so it can be regression-tested on
// every CI runner even though the current native status band is rendered by
// the Windows frontend.
func statusBandGeometry(height int) (statusY, contentBottom int) {
	statusY = height - statusBandHeight - statusBandBottomInset
	if statusY < 0 {
		statusY = 0
	}
	contentBottom = statusY - statusBandContentGap
	if contentBottom < 0 {
		contentBottom = 0
	}
	return statusY, contentBottom
}
